package karpenter

import (
	"bytes"
	"text/template"

	_ "embed"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-eks/sdk/v4/go/eks"
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed resources.yml
var resources string

type templateVars struct {
	ClusterName string
	Role        string
}

func ApplyResources(ctx *pulumi.Context, provider pulumi.ProviderResource, cluster *eks.Cluster, role *iam.Role, karpenterChart *helm.Chart) error {
	yamlContent := pulumi.All(cluster.EksCluster.Name(), role.Name, karpenterChart.URN()).
		ApplyT(func(args []any) (string, error) {
			vars := templateVars{ClusterName: args[0].(string), Role: args[1].(string)}
			tpl, err := template.New("resources").Parse(resources)
			if err != nil {
				return "", err
			}
			var buf bytes.Buffer
			if err := tpl.Execute(&buf, vars); err != nil {
				return "", err
			}
			return buf.String(), nil
		}).(pulumi.StringOutput)

	_, err := yaml.NewConfigGroup(
		ctx,
		"resources",
		&yaml.ConfigGroupArgs{
			Yaml: yamlContent,
		},
		pulumi.DependsOn([]pulumi.Resource{cluster}),
		pulumi.Provider(provider),
	)
	return err
}
