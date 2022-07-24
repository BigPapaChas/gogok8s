package clusters_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

type EKSMock struct{}

const (
	// AWS constants.
	east1 = "us-east-1"
	east2 = "us-east-2"
	west1 = "us-west-1"
	west2 = "us-west-2"

	// Property constants.
	caData = "ca-data"
)

//nolint:gochecknoglobals
var (
	// The testAccount is configured to scan 4 regions.
	testAccount = clusters.EKSAccount{
		Profile:    "dev",
		Regions:    []string{east1, east2, west1, west2},
		Name:       "Dev",
		Format:     "",
		ExtraUsers: []clusters.EKSUser{},
	}

	// us-west-2 clusters & descriptions.
	usWest2Clusters = &eks.ListClustersOutput{
		Clusters: []string{
			"foo",
			"bar",
		},
	}

	fooCluster = &eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn: aws.String("arn:aws:eks:us-west-2:012345678910:cluster/foo"),
			CertificateAuthority: &types.Certificate{
				Data: aws.String(base64.StdEncoding.EncodeToString([]byte(caData))),
			},
			Endpoint: aws.String("https://localhost:7777"),
			Name:     aws.String("foo"),
		},
	}
	barCluster = &eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn: aws.String("arn:aws:eks:us-west-2:012345678910:cluster/bar"),
			CertificateAuthority: &types.Certificate{
				Data: aws.String(base64.StdEncoding.EncodeToString([]byte(caData))),
			},
			Endpoint: aws.String("https://localhost:7777"),
			Name:     aws.String("bar"),
		},
	}

	// us-east-1 EKS clusters & descriptions.
	usEast1Clusters = &eks.ListClustersOutput{
		Clusters: []string{
			"staging",
			"production",
		},
	}

	stagingCluster = &eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn: aws.String("arn:aws:eks:us-east-1:012345678910:cluster/foo"),
			CertificateAuthority: &types.Certificate{
				Data: aws.String(base64.StdEncoding.EncodeToString([]byte(caData))),
			},
			Endpoint: aws.String("https://localhost:7777"),
			Name:     aws.String("staging"),
		},
	}
	productionCluster = &eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn: aws.String("arn:aws:eks:us-east-1:012345678910:cluster/bar"),
			CertificateAuthority: &types.Certificate{
				Data: aws.String(base64.StdEncoding.EncodeToString([]byte(caData))),
			},
			Endpoint: aws.String("https://localhost:7777"),
			Name:     aws.String("production"),
		},
	}

	// errors for tests.
	errClusterDoesNotExist = errors.New("cluster does not exist")
)

func (m EKSMock) ListClusters(
	ctx context.Context,
	params *eks.ListClustersInput,
	optFns ...func(*eks.Options),
) (*eks.ListClustersOutput, error) {
	opts := &eks.Options{}
	for _, optFn := range optFns {
		optFn(opts)
	}

	switch opts.Region {
	case east1:
		return usEast1Clusters, nil
	case west2:
		return usWest2Clusters, nil
	default:
		return &eks.ListClustersOutput{}, nil
	}
}

func (m EKSMock) DescribeCluster(
	ctx context.Context,
	params *eks.DescribeClusterInput,
	optFns ...func(*eks.Options),
) (*eks.DescribeClusterOutput, error) {
	opts := &eks.Options{}
	for _, optFn := range optFns {
		optFn(opts)
	}

	switch {
	case opts.Region == "us-west-2" && *params.Name == "foo":
		return fooCluster, nil
	case opts.Region == "us-west-2" && *params.Name == "bar":
		return barCluster, nil
	case opts.Region == "us-east-1" && *params.Name == "staging":
		return stagingCluster, nil
	case opts.Region == "us-east-1" && *params.Name == "production":
		return productionCluster, nil
	default:
		return nil, fmt.Errorf("%w: region=%s, cluster=%s", errClusterDoesNotExist, *params.Name, opts.Region)
	}
}

func TestEKSClusterScan(t *testing.T) {
	t.Parallel()

	client := EKSMock{}

	configs, errors := testAccount.ScanForClusters(client)
	if len(errors) > 0 {
		for _, err := range errors {
			t.Log(err)
		}

		t.Fatal("scanForClusters() experienced > 0 errors")
	}

	if len(configs) != 4 {
		t.Fatalf("scanForClusters() returned %d cluster configs, but expected %d", len(configs), 4)
	}

	for _, cfg := range configs {
		var cluster eks.DescribeClusterOutput

		switch cfg.Name {
		case "staging":
			cluster = *stagingCluster
		case "production":
			cluster = *productionCluster
		case "foo":
			cluster = *fooCluster
		case "bar":
			cluster = *barCluster
		default:
			t.Errorf("unknown cluster %s", cfg.Name)
		}

		switch {
		case cfg.Arn != *cluster.Cluster.Arn:
			t.Errorf("Cluster arn %s for cluster %s not equal to %s", cfg.Arn, cfg.Name, *cluster.Cluster.Arn)
		case !bytes.Equal(cfg.CertificateAuthorityData, []byte(caData)):
			t.Errorf("CertificateAuthorityData %s for cluster %s not equal to %s",
				string(cfg.CertificateAuthorityData),
				cfg.Name,
				caData)
		case cfg.Server != *cluster.Cluster.Endpoint:
			t.Errorf("Cluster endpoint %s for cluster %s not equal to %s",
				cfg.Server,
				cfg.Name,
				*cluster.Cluster.Endpoint)
		}
	}
}
