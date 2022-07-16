package kubecfg

import (
	"bytes"

	"github.com/BigPapaChas/gogok8s/internal/terminal"
	"k8s.io/client-go/tools/clientcmd/api"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

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
		!bytes.Equal(currentConfig.CertificateAuthorityData, cluster.Cluster.CertificateAuthorityData) {
		terminal.DiffModify(cluster.Name)
		if currentConfig.Server != cluster.Cluster.Server {
			terminal.DiffMinus(currentConfig.Server)
			terminal.DiffAdd(cluster.Cluster.Server)
		}
		if !bytes.Equal(currentConfig.CertificateAuthorityData, cluster.Cluster.CertificateAuthorityData) {
			terminal.DiffMinus("certificate-authority-data: <OMITTED>")
			terminal.DiffAdd("certificate-authority-data: <OMITTED>")
		}

		return true
	}

	return false
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
