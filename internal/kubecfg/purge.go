package kubecfg

import (
	"github.com/BigPapaChas/gogok8s/internal/terminal"
	"k8s.io/client-go/tools/clientcmd/api"
)

// The default clusters to ignore when purging a user's kubeconfig.
//nolint:gochecknoglobals
var defaultIgnoredClusters = map[string]struct{}{
	"docker-desktop":   {},
	"minikube":         {},
	"microk8s-cluster": {},
}

// The default users to ignore when purging a user's kubeconfig.
//nolint:gochecknoglobals
var defaultIgnoredUsers = map[string]struct{}{
	"docker-desktop": {},
	"minikube":       {},
	"admin":          {}, // used by microk8s
}

// The default contexts to ignore when purging a user's kubeconfig.
//nolint:gochecknoglobals
var defaultIgnoredContexts = map[string]struct{}{
	"docker-desktop": {},
	"minikube":       {},
	"microk8s":       {},
}

func purgeKubeConfig(patch *KubeConfigPatch, config *api.Config) {
	purgeClusters(patch, config)
	purgeUsers(patch, config)
	purgeContexts(patch, config)
}

func purgeClusters(patch *KubeConfigPatch, config *api.Config) {
	existingClusters := make(map[string]struct{})
	for _, cluster := range patch.Clusters {
		existingClusters[cluster.Name] = struct{}{}
	}

	var clustersToDelete []string

	for name := range config.Clusters {
		// Skip clusters ignored by default
		if _, ok := defaultIgnoredClusters[name]; ok {
			continue
		}

		if _, ok := existingClusters[name]; !ok {
			terminal.DiffMinus(name)
			clustersToDelete = append(clustersToDelete, name)
		}
	}

	for _, cluster := range clustersToDelete {
		delete(config.Clusters, cluster)
	}
}

func purgeUsers(patch *KubeConfigPatch, config *api.Config) {
	existingUsers := make(map[string]struct{})
	for _, user := range patch.Users {
		existingUsers[user.Name] = struct{}{}
	}

	var usersToDelete []string

	for name := range config.AuthInfos {
		// Skip users ignored by default
		if _, ok := defaultIgnoredUsers[name]; ok {
			continue
		}

		if _, ok := existingUsers[name]; !ok {
			usersToDelete = append(usersToDelete, name)
		}
	}

	for _, user := range usersToDelete {
		delete(config.AuthInfos, user)
	}
}

func purgeContexts(patch *KubeConfigPatch, config *api.Config) {
	existingContexts := make(map[string]struct{})
	for _, context := range patch.Contexts {
		existingContexts[context.Name] = struct{}{}
	}

	var contextsToDelete []string

	for name := range config.Contexts {
		// Skip contexts ignored by default
		if _, ok := defaultIgnoredContexts[name]; ok {
			continue
		}

		if _, ok := existingContexts[name]; !ok {
			contextsToDelete = append(contextsToDelete, name)
		}
	}

	for _, context := range contextsToDelete {
		delete(config.Contexts, context)
	}
}
