package security

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type IAMResources struct {
	ClusterRole   *iam.Role
	NodeGroupRole *iam.Role
	KarpenterRole *iam.Role
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
			"Name": pulumi.String(clusterName + "-cluster-role"),
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
			"Name": pulumi.String(clusterName + "-node-role"),
		},
	})
	if err != nil {
		return nil, err
	}

	nodeGroupPolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
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

	return &IAMResources{
		ClusterRole:   clusterRole,
		NodeGroupRole: nodeGroupRole,
		KarpenterRole: nil, // Will be created separately
	}, nil
}

func CreateKarpenterRole(ctx *pulumi.Context, clusterName string, oidcProviderArn pulumi.StringOutput, oidcProviderUrl pulumi.StringOutput, namespace string) (*iam.Role, error) {
	karpenterAssumeRolePolicy := pulumi.
		All(oidcProviderArn, oidcProviderUrl).
		ApplyT(func(args []any) string {
			providerArn := args[0].(string)
			providerUrl := args[1].(string)
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
		}`, providerArn, providerUrl, namespace, providerUrl)
		}).(pulumi.StringOutput)

	karpenterRole, err := iam.NewRole(ctx, clusterName+"-karpenter-role", &iam.RoleArgs{
		AssumeRolePolicy: karpenterAssumeRolePolicy,
		Tags: pulumi.StringMap{
			"Name": pulumi.String(clusterName + "-karpenter-role"),
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
					"ssm:GetParameter"
				],
				"Resource": "*"
			}
		]
	}`

	karpenterPolicy, err := iam.NewPolicy(ctx, clusterName+"-karpenter-policy", &iam.PolicyArgs{
		Policy: pulumi.String(karpenterPolicyDocument),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(clusterName + "-karpenter-policy"),
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
