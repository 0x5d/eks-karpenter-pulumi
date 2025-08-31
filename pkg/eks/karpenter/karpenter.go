package karpenter

import (
	"github.com/pulumi/pulumi-eks/sdk/v4/go/eks"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DeployChart(ctx *pulumi.Context, cluster *eks.Cluster) (*helm.Chart, error) {
	namespace := "karpenter"
	ns, err := corev1.NewNamespace(ctx, namespace, &corev1.NamespaceArgs{
		Metadata: &v1.ObjectMetaArgs{
			Name: pulumi.StringPtr(namespace),
		},
	})
	if err != nil {
		return nil, err
	}
	limits := pulumi.Map{
		"cpu":    pulumi.String("1"),
		"memory": pulumi.String("1Gi"),
	}
	return helm.NewChart(ctx, "karpenter", &helm.ChartArgs{
		Chart:     pulumi.String("oci://public.ecr.aws/karpenter/karpenter"),
		Version:   pulumi.String("1.6.3"),
		Namespace: ns.Metadata.Name(),
		Values: pulumi.Map{
			"settings": pulumi.Map{
				"clusterName":       cluster.EksCluster.Name(),
				"interruptionQueue": cluster.EksCluster.Name(),
			},
			"controller": pulumi.Map{
				"resources": pulumi.Map{
					"requests": limits,
					"limits":   limits,
				},
			},
		},
	})
}
