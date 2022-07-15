package clusters

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/BigPapaChas/gogok8s/internal/kubecfg"
)

type EKSAccount struct {
	Profile    string    `yaml:"profile"`
	Regions    []string  `yaml:"regions"`
	Name       string    `yaml:"name"`
	Format     string    `yaml:"format"`
	ExtraUsers []EKSUser `yaml:"extraUsers,omitempty"`
}

type EKSUser struct {
	Name    string `yaml:"name"`
	Profile string `yaml:"profile"`
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

const (
	defaultTimeout = time.Millisecond * 30000
	defaultFormat  = "${name}.${region}.${clusterName}"
)

func (a *EKSAccount) GenerateKubeConfigPatch() (*kubecfg.KubeConfigPatch, []error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(a.Profile))
	if err != nil {
		return nil, []error{err}
	}

	var allClusters []*EKSCluster

	var allErrors []error

	resultChan := make(chan DescribeRegionResult, len(a.Regions))
	client := eks.NewFromConfig(cfg)

	for _, region := range a.Regions {
		go func(eksClient *eks.Client, eksRegion string) {
			clusterNames, err := listClusters(eksClient, eksRegion)
			if err != nil {
				resultChan <- DescribeRegionResult{
					Errors: []error{fmt.Errorf("account='%s' region='%s': %w", a.Name, eksRegion, err)},
				}

				return
			}

			clusters, errors := getClusters(eksClient, clusterNames, eksRegion)
			resultChan <- DescribeRegionResult{
				Clusters: clusters,
				Errors:   errors,
			}
		}(client, region)
	}

	for range a.Regions {
		result := <-resultChan
		allClusters = append(allClusters, result.Clusters...)
		allErrors = append(allErrors, result.Errors...)
	}

	return a.generateKubeConfigPatch(allClusters), allErrors
}

func (a *EKSAccount) generateKubeConfigPatch(clusters []*EKSCluster) *kubecfg.KubeConfigPatch {
	patch := &kubecfg.KubeConfigPatch{}

	for _, cluster := range clusters {
		var clusterName, userName, contextName string

		replacements := map[string]string{
			"${name}":        a.Name,
			"${region}":      cluster.Region,
			"${clusterName}": cluster.Name,
			"${clusterArn}":  cluster.Arn,
		}

		formattedName := formatName(a.Format, replacements)
		clusterName = formattedName
		userName = formattedName
		contextName = formattedName

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
				Exec: generateIAMAuthenticatorExecConfig(cluster, a.Profile),
			},
		})

		for _, user := range a.ExtraUsers {
			patch.Users = append(patch.Users, &v1.NamedAuthInfo{
				Name: userName + "." + user.Name,
				AuthInfo: v1.AuthInfo{
					Exec: generateIAMAuthenticatorExecConfig(cluster, user.Profile),
				},
			})
			patch.Contexts = append(patch.Contexts, &v1.NamedContext{
				Name: contextName + "." + user.Name,
				Context: v1.Context{
					Cluster:  clusterName,
					AuthInfo: userName + "." + user.Name,
				},
			})
		}

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

func generateIAMAuthenticatorExecConfig(cluster *EKSCluster, profile string) *v1.ExecConfig {
	return &v1.ExecConfig{
		Command: "aws-iam-authenticator",
		Args:    []string{"token", "-i", cluster.Name, "--region", cluster.Region},
		Env: []v1.ExecEnvVar{
			{
				Name:  "AWS_PROFILE",
				Value: profile,
			},
		},
		APIVersion: "client.authentication.k8s.io/v1beta1",
	}
}

func getClusters(client *eks.Client, clusterNames []string, region string) ([]*EKSCluster, []error) {
	var clusters []*EKSCluster

	var errors []error

	resultChan := make(chan DescribeClusterResult, len(clusterNames))

	for _, clusterName := range clusterNames {
		go func(client *eks.Client, clusterName, region string) {
			description, err := describeCluster(client, clusterName, region)
			if err != nil {
				resultChan <- DescribeClusterResult{Error: err}

				return
			}

			decodedCertData, _ := base64.StdEncoding.DecodeString(*description.Cluster.CertificateAuthority.Data)

			resultChan <- DescribeClusterResult{
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
		result := <-resultChan
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

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	output, err := client.DescribeCluster(ctx, params, func(o *eks.Options) {
		o.Region = region
	})
	if err != nil {
		return nil, fmt.Errorf("cluster=%s, region=%s: %w", clusterName, region, err)
	}

	return output, nil
}

func listClusters(client *eks.Client, region string) ([]string, error) {
	params := &eks.ListClustersInput{}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	output, err := client.ListClusters(ctx, params, func(o *eks.Options) {
		o.Region = region
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
	}

	return output.Clusters, nil
}

func formatName(format string, replacements map[string]string) string {
	// Use default format if an empty format was passed
	if format == "" {
		format = defaultFormat
	}

	for old, replacement := range replacements {
		format = strings.ReplaceAll(format, old, replacement)
	}

	return format
}
