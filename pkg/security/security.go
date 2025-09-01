package security

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type IAMResources struct {
	ClusterRole       *iam.Role
	NodeGroupRole     *iam.Role
	KarpenterNodeRole *iam.Role
}

func CreateIAMRoles(ctx *pulumi.Context, clusterName string) (*IAMResources, error) {
	clusterAssumeRolePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "eks.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	clusterRole, err := iam.NewRole(ctx, clusterName+"-cluster-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(clusterAssumeRolePolicy),
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(clusterName + "-cluster-role"),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, clusterName+"-cluster-policy", &iam.RolePolicyAttachmentArgs{
		Role:      clusterRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
	})
	if err != nil {
		return nil, err
	}

	nodeGroupAssumeRolePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	nodeGroupRole, err := iam.NewRole(ctx, clusterName+"-node-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(nodeGroupAssumeRolePolicy),
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(clusterName + "-node-role"),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	nodeGroupPolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
	}

	for i, policy := range nodeGroupPolicies {
		_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-node-policy-%d", clusterName, i), &iam.RolePolicyAttachmentArgs{
			Role:      nodeGroupRole.Name,
			PolicyArn: pulumi.String(policy),
		})
		if err != nil {
			return nil, err
		}
	}

	karpenterNodeRole, err := CreateKarpenterNodeRole(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	return &IAMResources{
		ClusterRole:       clusterRole,
		NodeGroupRole:     nodeGroupRole,
		KarpenterNodeRole: karpenterNodeRole,
	}, nil
}

func CreateKarpenterRole(ctx *pulumi.Context, clusterName string, oidcProviderArn pulumi.StringOutput, oidcProviderUrl pulumi.StringOutput, namespace string) (*iam.Role, error) {
	karpenterAssumeRolePolicy := pulumi.
		All(oidcProviderArn, oidcProviderUrl).
		ApplyT(func(args []any) string {
			providerArn := args[0].(string)
			providerUrl := args[1].(string)
			// Remove https:// prefix if present for the condition
			cleanProviderUrl := strings.TrimPrefix(providerUrl, "https://")
			return fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {
					"Federated": "%s"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"%s:sub": "system:serviceaccount:%s:karpenter",
						"%s:aud": "sts.amazonaws.com"
					}
				}
			}]
		}`, providerArn, cleanProviderUrl, namespace, cleanProviderUrl)
		}).(pulumi.StringOutput)

	karpenterRole, err := iam.NewRole(ctx, clusterName+"-karpenter-role", &iam.RoleArgs{
		AssumeRolePolicy: karpenterAssumeRolePolicy,
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(clusterName + "-karpenter-role"),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	karpenterPolicyDocument := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": [
					"ec2:CreateLaunchTemplate",
					"ec2:CreateFleet",
					"ec2:RunInstances",
					"ec2:CreateTags",
					"ec2:TerminateInstances",
					"ec2:DeleteLaunchTemplate",
					"ec2:DescribeLaunchTemplates",
					"ec2:DescribeInstances",
					"ec2:DescribeInstanceTypes",
					"ec2:DescribeInstanceTypeOfferings",
					"ec2:DescribeAvailabilityZones",
					"ec2:DescribeSpotPriceHistory",
					"ec2:DescribeImages",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeSubnets",
					"pricing:GetProducts"
				],
				"Resource": "*"
			},
			{
				"Effect": "Allow",
				"Action": [
					"eks:DescribeCluster",
					"eks:DescribeNodegroup"
				],
				"Resource": "*"
			},
			{
				"Effect": "Allow",
				"Action": [
					"iam:PassRole"
				],
				"Resource": "*",
				"Condition": {
					"StringEquals": {
						"iam:PassedToService": "ec2.amazonaws.com"
					}
				}
			},
			{
				"Effect": "Allow",
				"Action": [
					"iam:GetInstanceProfile",
					"iam:CreateInstanceProfile",
					"iam:DeleteInstanceProfile",
					"iam:AddRoleToInstanceProfile",
					"iam:RemoveRoleFromInstanceProfile",
					"iam:TagInstanceProfile",
					"iam:UntagInstanceProfile"
				],
				"Resource": "*"
			},
			{
				"Effect": "Allow",
				"Action": [
					"ssm:GetParameter"
				],
				"Resource": "*"
			}
		]
	}`

	karpenterPolicy, err := iam.NewPolicy(ctx, clusterName+"-karpenter-policy", &iam.PolicyArgs{
		Policy: pulumi.String(karpenterPolicyDocument),
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(clusterName + "-karpenter-policy"),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, clusterName+"-karpenter-policy-attachment", &iam.RolePolicyAttachmentArgs{
		Role:      karpenterRole.Name,
		PolicyArn: karpenterPolicy.Arn,
	})
	if err != nil {
		return nil, err
	}

	return karpenterRole, nil
}

func CreateKarpenterNodeRole(ctx *pulumi.Context, clusterName string) (*iam.Role, error) {
	nodeAssumeRolePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	nodeRoleName := clusterName + "-karpenter-node-role"
	nodeRole, err := iam.NewRole(ctx, nodeRoleName, &iam.RoleArgs{
		Name:             pulumi.String(nodeRoleName),
		AssumeRolePolicy: pulumi.String(nodeAssumeRolePolicy),
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(nodeRoleName),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	// Attach the required managed policies for Karpenter nodes
	nodePolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
	}

	for i, policy := range nodePolicies {
		_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-karpenter-node-policy-%d", clusterName, i), &iam.RolePolicyAttachmentArgs{
			Role:      nodeRole.Name,
			PolicyArn: pulumi.String(policy),
		})
		if err != nil {
			return nil, err
		}
	}

	// Create instance profile for the node role
	instanceProfile, err := iam.NewInstanceProfile(ctx, nodeRoleName+"-profile", &iam.InstanceProfileArgs{
		Name: pulumi.String(nodeRoleName),
		Role: nodeRole.Name,
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(nodeRoleName + "-profile"),
			"managed-by": pulumi.String("pulumi"),
		},
	})
	if err != nil {
		return nil, err
	}

	// Export the instance profile name for use by Karpenter
	ctx.Export("karpenterNodeInstanceProfile", instanceProfile.Name)

	return nodeRole, nil
}
