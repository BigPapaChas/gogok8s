package clusters

import (
	"context"
	"encoding/base64"
	"fmt"
	"gogok8s/internal/kubecfg"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type EKSAccount struct {
	Profile string   `yaml:"profile"`
	Regions []string `yaml:"regions"`
	Name    string   `yaml:"name"`
	Format  string   `yaml:"format"`
}

type EKSCluster struct {
	Name                     string
	Region                   string
	Server                   string
	CertificateAuthorityData []byte
	Arn                      string
}

type DescribeRegionResult struct {
	Clusters []*EKSCluster
	Errors   []error
}

type DescribeClusterResult struct {
	Cluster *EKSCluster
	Error   error
}

func (a *EKSAccount) GenerateKubeConfigPatch() (*kubecfg.KubeConfigPatch, []error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(a.Profile))
	if err != nil {
		return nil, []error{err}
	}

	var allClusters []*EKSCluster
	var allErrors []error
	c := make(chan DescribeRegionResult, len(a.Regions))
	client := eks.NewFromConfig(cfg)
	for _, region := range a.Regions {
		go func(eksClient *eks.Client, eksRegion string) {
			clusterNames, err := listClusters(eksClient, eksRegion)
			if err != nil {
				c <- DescribeRegionResult{
					Errors: []error{fmt.Errorf("account='%s' region='%s': %v", a.Name, eksRegion, err)},
				}
				return
			}

			clusters, errors := getClusters(eksClient, clusterNames, eksRegion)
			c <- DescribeRegionResult{
				Clusters: clusters,
				Errors:   errors,
			}
		}(client, region)
	}

	for range a.Regions {
		result := <-c
		allClusters = append(allClusters, result.Clusters...)
		allErrors = append(allErrors, result.Errors...)
	}

	patch := a.generateKubeConfigPatch(allClusters)
	return patch, allErrors
}

func (a *EKSAccount) generateKubeConfigPatch(clusters []*EKSCluster) *kubecfg.KubeConfigPatch {
	patch := &kubecfg.KubeConfigPatch{}

	for _, cluster := range clusters {
		var clusterName, userName, contextName string
		// TODO: Create a more formal format parsing
		if a.Format != "" {
			clusterName = strings.Replace(a.Format, "${cluster}", cluster.Name, 1)
			userName = strings.Replace(a.Format, "${cluster}", cluster.Name, 1)
			contextName = strings.Replace(a.Format, "${cluster}", cluster.Name, 1)
		} else {
			clusterName = fmt.Sprintf("%s", cluster.Arn)
			userName = fmt.Sprintf("%s", cluster.Arn)
			contextName = fmt.Sprintf("%s", cluster.Arn)
		}

		patch.Clusters = append(patch.Clusters, &v1.NamedCluster{
			Name: clusterName,
			Cluster: v1.Cluster{
				Server:                   cluster.Server,
				CertificateAuthorityData: cluster.CertificateAuthorityData,
			},
		})

		patch.Users = append(patch.Users, &v1.NamedAuthInfo{
			Name: userName,
			AuthInfo: v1.AuthInfo{
				Exec: a.generateIAMAuthenticatorExecConfig(cluster),
			},
		})

		patch.Contexts = append(patch.Contexts, &v1.NamedContext{
			Name: contextName,
			Context: v1.Context{
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		})
	}
	return patch
}

func (a *EKSAccount) generateIAMAuthenticatorExecConfig(cluster *EKSCluster) *v1.ExecConfig {
	return &v1.ExecConfig{
		Command: "aws-iam-authenticator",
		Args:    []string{"token", "-i", cluster.Name, "--region", cluster.Region},
		Env: []v1.ExecEnvVar{
			{
				Name:  "AWS_PROFILE",
				Value: a.Profile,
			},
		},
		APIVersion: "client.authentication.k8s.io/v1alpha1",
	}
}

func getClusters(client *eks.Client, clusterNames []string, region string) ([]*EKSCluster, []error) {
	var clusters []*EKSCluster
	var errors []error

	c := make(chan DescribeClusterResult, len(clusterNames))
	for _, clusterName := range clusterNames {
		go func(client *eks.Client, clusterName, region string) {
			description, err := describeCluster(client, clusterName, region)
			if err != nil {
				c <- DescribeClusterResult{Error: err}
				return
			}
			decodedCertData, _ := base64.StdEncoding.DecodeString(*description.Cluster.CertificateAuthority.Data)
			c <- DescribeClusterResult{
				Cluster: &EKSCluster{
					Name:                     clusterName,
					Region:                   region,
					Server:                   *description.Cluster.Endpoint,
					CertificateAuthorityData: decodedCertData,
					Arn:                      *description.Cluster.Arn,
				},
				Error: err,
			}
		}(client, clusterName, region)
	}

	for range clusterNames {
		result := <-c
		if result.Error != nil {
			// DescribeCluster encountered an error, add to list of errors and continue to next result
			errors = append(errors, result.Error)
			continue
		}
		clusters = append(clusters, result.Cluster)
	}

	return clusters, errors
}

func describeCluster(client *eks.Client, clusterName, region string) (*eks.DescribeClusterOutput, error) {
	params := &eks.DescribeClusterInput{Name: &clusterName}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	output, err := client.DescribeCluster(ctx, params, func(o *eks.Options) {
		o.Region = region
	})
	if err != nil {
		return nil, fmt.Errorf("cluster=%s, region=%s: %v", clusterName, region, err)
	}

	return output, nil
}

func listClusters(client *eks.Client, region string) ([]string, error) {
	params := &eks.ListClustersInput{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	output, err := client.ListClusters(ctx, params, func(o *eks.Options) {
		o.Region = region
	})

	if err != nil {
		return nil, err
	}
	return output.Clusters, nil
}
