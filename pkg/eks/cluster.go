package eks

import (
	"eks-karpenter/pkg/config"
	"eks-karpenter/pkg/security"
	"eks-karpenter/pkg/vpc"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-eks/sdk/v4/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EKSResources struct {
	Cluster   *eks.Cluster
	NodeGroup *eks.NodeGroupV2
}

func CreateEKSCluster(ctx *pulumi.Context, cfg *config.Config, vpcResources *vpc.VPCResources, iamResources *security.IAMResources) (*EKSResources, error) {
	subnetIds := make(pulumi.StringArray, len(vpcResources.PublicSubnets))
	for i, subnet := range vpcResources.PublicSubnets {
		subnetIds[i] = subnet.ID()
	}

	// Create a security group for nodes with outbound HTTPS access
	nodeSecurityGroup, err := ec2.NewSecurityGroup(ctx, cfg.ClusterName+"-node-sg", &ec2.SecurityGroupArgs{
		VpcId:       vpcResources.VPC.ID(),
		Description: pulumi.String("Security group for EKS node group with HTTPS egress"),
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(443),
				ToPort:     pulumi.Int(443),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name":                   pulumi.String(cfg.ClusterName + "-node-sg"),
			"karpenter.sh/discovery": pulumi.String(cfg.ClusterName),
		},
	})
	if err != nil {
		return nil, err
	}

	cluster, err := eks.NewCluster(ctx, cfg.ClusterName, &eks.ClusterArgs{
		// Name:               pulumi.String(cfg.ClusterName),
		Version:            pulumi.String(cfg.KubernetesVersion),
		ServiceRole:        iamResources.ClusterRole,
		SubnetIds:          subnetIds,
		CreateOidcProvider: pulumi.Bool(true),
		VpcId:              vpcResources.VPC.ID(),
	})
	if err != nil {
		return nil, err
	}

	nodeGroup, err := eks.NewNodeGroupV2(ctx, cfg.ClusterName+"-nodes", &eks.NodeGroupV2Args{
		Cluster:                 cluster,
		InstanceType:            pulumi.String("t3.medium"),
		DesiredCapacity:         pulumi.Int(cfg.NodeCount),
		MinSize:                 pulumi.Int(cfg.MinSize),
		MaxSize:                 pulumi.Int(cfg.MaxSize),
		ExtraNodeSecurityGroups: ec2.SecurityGroupArray{nodeSecurityGroup},
	})
	if err != nil {
		return nil, err
	}

	return &EKSResources{
		Cluster:   cluster,
		NodeGroup: nodeGroup,
	}, nil
}
