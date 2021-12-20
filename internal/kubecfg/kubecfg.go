package kubecfg

import (
	"bytes"
	"gogok8s/internal/terminal"
	"os"
	"path"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
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
	return clientcmd.LoadFromFile(filename)
}

func Write(config *api.Config) error {
	filename, err := getKubeConfigFilePath()
	if err != nil {
		return err
	}

	return writeKubeConfigToFile(config, filename)
}

func ApplyPatch(patch *KubeConfigPatch, config *api.Config) {
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
}

func applyClusterChanges(config *api.Config, cluster *v1.NamedCluster) {
	compareClusterChanges(config, cluster)
	if _, ok := config.Clusters[cluster.Name]; !ok {
		config.Clusters[cluster.Name] = &api.Cluster{
			Server:                   cluster.Cluster.Server,
			CertificateAuthorityData: cluster.Cluster.CertificateAuthorityData,
		}
	} else {
		config.Clusters[cluster.Name].Server = cluster.Cluster.Server
		config.Clusters[cluster.Name].CertificateAuthorityData = cluster.Cluster.CertificateAuthorityData
	}
}

func applyUserChanges(config *api.Config, user *v1.NamedAuthInfo) {
	if _, ok := config.AuthInfos[user.Name]; !ok {
		config.AuthInfos[user.Name] = &api.AuthInfo{
			Exec: &api.ExecConfig{
				Command:    user.AuthInfo.Exec.Command,
				Args:       user.AuthInfo.Exec.Args,
				Env:        convertExecEnvVar(user.AuthInfo.Exec.Env),
				APIVersion: user.AuthInfo.Exec.APIVersion,
			},
		}
	} else {
		config.AuthInfos[user.Name].Exec.Command = user.AuthInfo.Exec.Command
		config.AuthInfos[user.Name].Exec.Args = user.AuthInfo.Exec.Args
		config.AuthInfos[user.Name].Exec.Env = convertExecEnvVar(user.AuthInfo.Exec.Env)
		config.AuthInfos[user.Name].Exec.APIVersion = user.AuthInfo.Exec.APIVersion
	}
}

func applyContextChanges(config *api.Config, context *v1.NamedContext) {
	if _, ok := config.Contexts[context.Name]; !ok {
		config.Contexts[context.Name] = &api.Context{
			Cluster:  context.Context.Cluster,
			AuthInfo: context.Context.AuthInfo,
		}
	} else {
		config.Contexts[context.Name].Cluster = context.Context.Cluster
		config.Contexts[context.Name].AuthInfo = context.Context.AuthInfo
	}
}

func compareClusterChanges(config *api.Config, cluster *v1.NamedCluster) bool {
	if currentConfig, ok := config.Clusters[cluster.Name]; !ok {
		terminal.DiffAdd(cluster.Name)
		return true
	} else if currentConfig.Server != cluster.Cluster.Server ||
		bytes.Compare(currentConfig.CertificateAuthorityData, cluster.Cluster.CertificateAuthorityData) != 0 {
		terminal.DiffModify(cluster.Name)
		if currentConfig.Server != cluster.Cluster.Server {
			terminal.DiffMinus(currentConfig.Server)
			terminal.DiffAdd(cluster.Cluster.Server)
		}
		if bytes.Compare(currentConfig.CertificateAuthorityData, cluster.Cluster.CertificateAuthorityData) != 0 {
			terminal.DiffMinus("certificate-authority-data: <OMITTED>")
			terminal.DiffAdd("certificate-authority-data: <OMITTED>")
		}
		return true
	}
	return false
}

func compareUserChanges(config *api.Config, user *v1.NamedAuthInfo) bool {
	if currentConfig, ok := config.AuthInfos[user.Name]; !ok {
		return true
	} else if currentConfig.Exec.APIVersion != user.AuthInfo.Exec.APIVersion ||
		currentConfig.Exec.Command != user.AuthInfo.Exec.Command ||
		!execArgsEqual(currentConfig.Exec.Args, user.AuthInfo.Exec.Args) ||
		!execEnvEqual(currentConfig.Exec.Env, user.AuthInfo.Exec.Env) {
		return true
	}
	return false
}

func compareContextChanges(config *api.Config, context *v1.NamedContext) bool {
	if currentConfig, ok := config.Contexts[context.Name]; !ok {
		return true
	} else if currentConfig.AuthInfo != context.Context.AuthInfo ||
		currentConfig.Cluster != context.Context.Cluster {
		return true
	}
	return false
}

func execArgsEqual(args1, args2 []string) bool {
	if len(args1) != len(args2) {
		return false
	}
	for i, v := range args1 {
		if v != args2[i] {
			return false
		}
	}
	return true
}

func execEnvEqual(env1 []api.ExecEnvVar, env2 []v1.ExecEnvVar) bool {
	if len(env1) != len(env2) {
		return false
	}

	envMap := make(map[string]string)
	for _, env := range env1 {
		envMap[env.Name] = env.Value
	}

	for _, env := range env2 {
		if v, ok := envMap[env.Name]; !ok {
			return false
		} else if v != env.Value {
			return false
		}
	}
	return true
}

func convertExecEnvVar(envVars []v1.ExecEnvVar) []api.ExecEnvVar {
	var convertedExecEnvVars []api.ExecEnvVar
	for _, envVar := range envVars {
		convertedExecEnvVars = append(convertedExecEnvVars, api.ExecEnvVar{
			Name:  envVar.Name,
			Value: envVar.Value,
		})
	}
	return convertedExecEnvVars
}

func writeKubeConfigToFile(config *api.Config, filename string) error {
	return clientcmd.WriteToFile(*config, filename)
}

func getKubeConfigFilePath() (string, error) {
	var filename string
	filename, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		filename = path.Join(homedir, ".kube", "config")
	}
	return filename, nil
}
