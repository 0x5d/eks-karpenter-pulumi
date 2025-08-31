package vpc

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VPCResources struct {
	VPC             *ec2.Vpc
	PublicSubnets   []*ec2.Subnet
	InternetGateway *ec2.InternetGateway
	RouteTable      *ec2.RouteTable
}

func CreateVPC(ctx *pulumi.Context, clusterName string) (*VPCResources, error) {
	vpc, err := ec2.NewVpc(ctx, clusterName+"-vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String("10.0.0.0/16"),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(clusterName + "-vpc"),
		},
	})
	if err != nil {
		return nil, err
	}

	igw, err := ec2.NewInternetGateway(ctx, clusterName+"-igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(clusterName + "-igw"),
		},
	})
	if err != nil {
		return nil, err
	}

	routeTable, err := ec2.NewRouteTable(ctx, clusterName+"-public-rt", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String(clusterName + "-public-rt"),
		},
	})
	if err != nil {
		return nil, err
	}

	azs := []string{"a", "b", "c"}
	var subnets []*ec2.Subnet

	for i, az := range azs {
		subnet, err := ec2.NewSubnet(ctx, clusterName+"-public-subnet-"+az, &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(fmt.Sprintf("10.0.%d.0/24", i+1)),
			AvailabilityZone:    pulumi.String(fmt.Sprintf("us-east-2%s", az)),
			MapPublicIpOnLaunch: pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name":                                 pulumi.String(clusterName + "-public-subnet-" + az),
				"kubernetes.io/role/elb":               pulumi.String("1"),
				"kubernetes.io/cluster/" + clusterName: pulumi.String("owned"),
			},
		})
		if err != nil {
			return nil, err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, clusterName+"-public-rta-"+az, &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet.ID(),
			RouteTableId: routeTable.ID(),
		})
		if err != nil {
			return nil, err
		}

		subnets = append(subnets, subnet)
	}

	return &VPCResources{
		VPC:             vpc,
		PublicSubnets:   subnets,
		InternetGateway: igw,
		RouteTable:      routeTable,
	}, nil
}
