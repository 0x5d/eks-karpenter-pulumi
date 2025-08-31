package security

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type IAMResources struct {
	ClusterRole   *iam.Role
	NodeGroupRole *iam.Role
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
	}, nil
}
