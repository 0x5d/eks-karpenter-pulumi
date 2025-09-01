package karpenter

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v4/go/eks"

	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DeployChart(ctx *pulumi.Context, provider pulumi.ProviderResource, cluster *eks.Cluster, karpenterRoleArn *iam.Role, namespace string) (*helm.Chart, error) {
	limits := pulumi.Map{
		"cpu":    pulumi.String("1"),
		"memory": pulumi.String("1Gi"),
	}
	return helm.NewChart(
		ctx,
		"karpenter",
		&helm.ChartArgs{
			Chart:     pulumi.String("oci://public.ecr.aws/karpenter/karpenter"),
			Version:   pulumi.String("1.6.3"),
			Namespace: pulumi.String(namespace),
			Values: pulumi.Map{
				"serviceAccount": pulumi.Map{
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
		pulumi.DependsOn([]pulumi.Resource{cluster, karpenterRoleArn}),
		pulumi.Provider(provider),
	)
}
