package clusters

import (
	"context"
	"encoding/base64"
	"fmt"
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

type EKSClusterConfig struct {
	Name                     string
	Region                   string
	Server                   string
	CertificateAuthorityData []byte
	Arn                      string
}

type describeEKSResult struct {
	Cluster EKSClusterConfig
	Error   error
}

type scanForClustersResult struct {
	Clusters []EKSClusterConfig
	Errors   []error
}

type EKSClusterAPI interface {
	ListClusters(
		ctx context.Context,
		params *eks.ListClustersInput,
		optFns ...func(*eks.Options),
	) (*eks.ListClustersOutput, error)
	DescribeCluster(
		ctx context.Context,
		params *eks.DescribeClusterInput,
		optFns ...func(*eks.Options),
	) (*eks.DescribeClusterOutput, error)
}

const (
	defaultTimeout = 30 * time.Second
	defaultFormat  = "${name}.${region}.${clusterName}"
)

func (a EKSAccount) GenerateKubeConfig() (*kubecfg.KubeConfigPatch, []error) {
	accountKubeConfig := &kubecfg.KubeConfigPatch{}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(a.Profile))
	if err != nil {
		return accountKubeConfig, []error{err}
	}

	client := eks.NewFromConfig(cfg)
	clusters, errors := a.ScanForClusters(client)

	for _, cluster := range clusters {
		patch := a.generateKubeConfigFromCluster(cluster)
		accountKubeConfig.Clusters = append(accountKubeConfig.Clusters, patch.Clusters...)
		accountKubeConfig.Users = append(accountKubeConfig.Users, patch.Users...)
		accountKubeConfig.Contexts = append(accountKubeConfig.Contexts, patch.Contexts...)
	}

	return accountKubeConfig, errors
}

func (a EKSAccount) PrettyName() string {
	return a.Name
}

func (a EKSAccount) generateKubeConfigFromCluster(cluster EKSClusterConfig) *kubecfg.KubeConfigPatch {
	patch := &kubecfg.KubeConfigPatch{}

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

	return patch
}

func (a EKSAccount) ScanForClusters(client EKSClusterAPI) ([]EKSClusterConfig, []error) {
	ch := make(chan scanForClustersResult, len(a.Regions))

	for _, region := range a.Regions {
		go scanForClustersInRegion(region, client, ch)
	}

	var clusters []EKSClusterConfig

	var errors []error

	for range a.Regions {
		result := <-ch
		clusters = append(clusters, result.Clusters...)
		errors = append(errors, result.Errors...)
	}

	return clusters, errors
}

func scanForClustersInRegion(region string, client EKSClusterAPI, ch chan scanForClustersResult) {
	clusterNames, err := listEKSClusters(client, region)
	if err != nil {
		ch <- scanForClustersResult{
			Errors: []error{fmt.Errorf("region='%s': %w", region, err)},
		}

		return
	}

	clusters, errors := getEKSClusterConfigs(client, clusterNames, region)
	ch <- scanForClustersResult{
		Clusters: clusters,
		Errors:   errors,
	}
}

func getEKSClusterConfigs(client EKSClusterAPI, clusterNames []string, region string) ([]EKSClusterConfig, []error) {
	var clusters []EKSClusterConfig

	var errors []error

	ch := make(chan describeEKSResult, len(clusters))

	for _, clusterName := range clusterNames {
		go getEKSClusterConfig(client, clusterName, region, ch)
	}

	for range clusterNames {
		result := <-ch
		if result.Error != nil {
			// DescribeCluster encountered an error, add to list of errors and continue to next result
			errors = append(errors, result.Error)

			continue
		}

		clusters = append(clusters, result.Cluster)
	}

	return clusters, errors
}

func getEKSClusterConfig(client EKSClusterAPI, clusterName, region string, ch chan describeEKSResult) {
	description, err := describeEKSCluster(client, clusterName, region)
	if err != nil {
		ch <- describeEKSResult{Error: err}

		return
	}

	decodedCertData, _ := base64.StdEncoding.DecodeString(*description.Cluster.CertificateAuthority.Data)

	ch <- describeEKSResult{
		Cluster: EKSClusterConfig{
			Name:                     *description.Cluster.Name,
			Region:                   region,
			Server:                   *description.Cluster.Endpoint,
			CertificateAuthorityData: decodedCertData,
			Arn:                      *description.Cluster.Arn,
		},
		Error: err,
	}
}

func describeEKSCluster(client EKSClusterAPI, clusterName, region string) (*eks.DescribeClusterOutput, error) {
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

func listEKSClusters(client EKSClusterAPI, region string) ([]string, error) {
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

func generateIAMAuthenticatorExecConfig(cluster EKSClusterConfig, profile string) *v1.ExecConfig {
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
