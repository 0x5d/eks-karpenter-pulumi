package main

import (
	"eks-karpenter/pkg/config"
	"eks-karpenter/pkg/eks"
	"eks-karpenter/pkg/eks/karpenter"
	"eks-karpenter/pkg/security"
	"eks-karpenter/pkg/vpc"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.GetConfig(ctx)
		karpenterNamespace := "kube-system"

		vpcResources, err := vpc.CreateVPC(ctx, cfg.ClusterName)
		if err != nil {
			return err
		}

		iamResources, err := security.CreateIAMRoles(ctx, cfg.ClusterName)
		if err != nil {
			return err
		}

		eksResources, err := eks.CreateEKSCluster(ctx, cfg, vpcResources, iamResources)
		if err != nil {
			return err
		}

		karpenterRole, err := security.CreateKarpenterRole(ctx, cfg.ClusterName,
			eksResources.Cluster.Core.OidcProvider().Arn(),
			eksResources.Cluster.Core.OidcProvider().Url(),
			karpenterNamespace)
		if err != nil {
			return err
		}

		k8sProvider, err := kubernetes.NewProvider(
			ctx,
			"cluster",
			&kubernetes.ProviderArgs{
				Kubeconfig: eksResources.Cluster.KubeconfigJson.ToStringOutput(),
			},
			pulumi.DependsOn([]pulumi.Resource{eksResources.Cluster}),
		)
		if err != nil {
			return err
		}

		_, err = karpenter.DeployChart(ctx, k8sProvider, eksResources.Cluster, karpenterRole, karpenterNamespace)
		if err != nil {
			return err
		}

		ctx.Export("clusterArn", eksResources.Cluster.Core.Cluster().Arn())
		ctx.Export("clusterEndpoint", eksResources.Cluster.Core.Endpoint())
		ctx.Export("clusterName", eksResources.Cluster.Core.Cluster().Name())
		ctx.Export("clusterVersion", eksResources.Cluster.Core.Cluster().Version())
		ctx.Export("kubeconfig", pulumi.ToSecret(eksResources.Cluster.Kubeconfig))
		ctx.Export("vpcId", vpcResources.VPC.ID())

		return nil
	})
}
