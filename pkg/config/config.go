package config

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Config struct {
	ClusterName       string
	Region            string
	NodeCount         int
	KubernetesVersion string
}

func GetConfig(ctx *pulumi.Context) *Config {
	cfg := config.New(ctx, "")

	clusterName := cfg.Get("clusterName")
	if clusterName == "" {
		clusterName = "eks-cluster"
	}

	region := cfg.Get("region")
	if region == "" {
		region = "us-east-2"
	}

	nodeCount := cfg.GetInt("nodeCount")
	if nodeCount == 0 {
		nodeCount = 3
	}

	kubernetesVersion := cfg.Get("kubernetesVersion")
	if kubernetesVersion == "" {
		kubernetesVersion = "1.33"
	}

	return &Config{
		ClusterName:       clusterName,
		Region:            region,
		NodeCount:         nodeCount,
		KubernetesVersion: kubernetesVersion,
	}
}
