package clusters

import (
	"github.com/BigPapaChas/gogok8s/internal/kubecfg"
)

type ClusterAccount interface {
	GenerateKubeConfig() (*kubecfg.KubeConfigPatch, []error)
	PrettyName() string
}
