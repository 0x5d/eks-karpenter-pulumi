package karpenter

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v4/go/eks"

	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DeployChart(ctx *pulumi.Context, provider pulumi.ProviderResource, cluster *eks.Cluster, karpenterRoleArn *iam.Role, namespace string) (*helm.Release, error) {
	limits := pulumi.Map{
		"cpu":    pulumi.String("1"),
		"memory": pulumi.String("1Gi"),
	}

	karpenterVersion := pulumi.String("1.6.3")
	crds, err := helm.NewRelease(
		ctx,
		"karpenter-crds",
		&helm.ReleaseArgs{
			Chart:     pulumi.String("oci://public.ecr.aws/karpenter/karpenter-crd"),
			Version:   karpenterVersion,
			Namespace: pulumi.String(namespace),
			SkipCrds:  pulumi.Bool(false),
		},
		pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{cluster}),
	)
	if err != nil {
		return nil, fmt.Errorf("error installing karpenter crds: %w", err)
	}

	return helm.NewRelease(
		ctx,
		"karpenter",
		&helm.ReleaseArgs{
			Chart:     pulumi.String("oci://public.ecr.aws/karpenter/karpenter"),
			Version:   karpenterVersion,
			Namespace: pulumi.String(namespace),
			SkipCrds:  pulumi.BoolPtr(true),
			Values: pulumi.Map{
				"serviceAccount": pulumi.Map{
					"name": pulumi.String("karpenter"),
					"annotations": pulumi.Map{
						"eks.amazonaws.com/role-arn": karpenterRoleArn.Arn,
					},
				},
				"settings": pulumi.Map{
					"clusterName":     cluster.EksCluster.Name(),
					"clusterEndpoint": cluster.EksCluster.Endpoint(),
					// Enable if we decide to use spot instances - EC2 publishes eviction warnings here
					// and Karpenter is able to react beforehand (e.g. by spinning up a new VM).
					// "interruptionQueue": <SQS queue>,
				},
				"controller": pulumi.Map{
					"resources": pulumi.Map{
						"requests": limits,
						"limits":   limits,
					},
				},
			},
		},
		pulumi.DependsOn([]pulumi.Resource{cluster, karpenterRoleArn, crds}),
		pulumi.Provider(provider),
	)
}
