package eks

import (
	"eks-karpenter/pkg/config"
	"eks-karpenter/pkg/security"
	"eks-karpenter/pkg/vpc"

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

	cluster, err := eks.NewCluster(ctx, cfg.ClusterName, &eks.ClusterArgs{
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
		Cluster:         cluster,
		InstanceType:    pulumi.String("t3.medium"),
		DesiredCapacity: pulumi.Int(cfg.NodeCount),
		MinSize:         pulumi.Int(1),
		MaxSize:         pulumi.Int(5),
	})
	if err != nil {
		return nil, err
	}

	return &EKSResources{
		Cluster:   cluster,
		NodeGroup: nodeGroup,
	}, nil
}
