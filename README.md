# Gogok8s

Gogok8s is a CLI tool that allows for easy management of kubeconfig files for EKS clusters. It can discover new clusters
in your accounts and will update your kubeconfig without needing to run `aws eks update-kubeconfig` each time a new
cluster is created in your accounts.

<p align="center"><img src="/img/demo.gif?raw=true" alt="gogok8s-demo"/></p>

# Installation

```bash
go install github.com/BigPapaChas/gogok8s/cmd/gogok8s
```

# Getting Started

Create your `.gogok8s.yaml` config file by using the `configure` command.

<p align="center"><img src="/img/gogok8s-configure.gif?raw=true" alt="gogok8s-configure-demo"/></p>
