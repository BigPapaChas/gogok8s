package kubecfg

import (
	"errors"
	"fmt"
	"os"
	"path"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

type KubeConfigPatch struct {
	Clusters []*v1.NamedCluster
	Users    []*v1.NamedAuthInfo
	Contexts []*v1.NamedContext
}

func LoadDefault() (*api.Config, error) {
	filename, err := getKubeConfigFilePath()
	if err != nil {
		return nil, err
	}

	return LoadFromFile(filename)
}

func LoadFromFile(filename string) (*api.Config, error) {
	cfg, err := clientcmd.LoadFromFile(filename)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return api.NewConfig(), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from file: %w", err)
	}

	return cfg, nil
}

func Write(config *api.Config) error {
	filename, err := getKubeConfigFilePath()
	if err != nil {
		return err
	}

	if err = clientcmd.WriteToFile(*config, filename); err != nil {
		return fmt.Errorf("failed to write kubeconfig to file: %w", err)
	}

	return nil
}

func ApplyPatch(patch *KubeConfigPatch, config *api.Config, purge bool) {
	if patch == nil {
		return
	}

	terminal.TextYellow("\nApplying changes to kubeconfig")

	for _, cluster := range patch.Clusters {
		applyClusterChanges(config, cluster)
	}

	for _, user := range patch.Users {
		applyUserChanges(config, user)
	}

	for _, context := range patch.Contexts {
		applyContextChanges(config, context)
	}

	if purge {
		purgeKubeConfig(patch, config)
	}
}

func getKubeConfigFilePath() (string, error) {
	filename, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to find user home directory: %w", err)
		}

		filename = path.Join(homedir, ".kube", "config")
	}

	return filename, nil
}
