# Gogok8s

Gogok8s is a CLI tool that allows for easy management of kubeconfig files for EKS clusters. It can discover new clusters
in your accounts and will update your kubeconfig without needing to run `aws eks update-kubeconfig` each time a new
cluster is created in your accounts.

<p align="center"><img src="/img/demo.gif?raw=true" alt="gogok8s-demo"/></p>

---

# Installing & Updating

```bash
go install github.com/BigPapaChas/gogok8s/cmd/gogok8s@latest
```

# Getting Started

Create your `.gogok8s.yaml` config file by using the `configure` command.

<p align="center"><img src="/img/gogok8s-configure.gif?raw=true" alt="gogok8s-configure-demo"/></p>

## Example Config

An example gogok8s config might look something like this:

```yaml
accounts:
  - name: Dev
    profile: dev-admin
    regions:
      - us-east-1
      - us-east-2
    format: ""
  - name: Staging
    profile: staging-admin
    regions:
      - us-east-1
      - us-east-2
      - us-west-1
      - us-west-2
      - eu-west-1
    format: "${name}.${clusterArn}"
  - name: Prod
    profile: prod-admin-readonly
    regions:
      - us-east-1
      - us-east-2
      - us-west-1
      - us-west-2
      - eu-west-1
    format: "prod.${region}.${clusterName}"
    extraUsers:
      - name: admin
        profile: prod-admin-write
```

Each entry in `accounts` can have the following fields:
- `name` - A convenient name you wish to give for this AWS account
- `profile` - The AWS profile name used to list & describe EKS clusters
- `regions` - The list of AWS regions that will be searched for EKS clusters
- `format` - The format of the kubeconfig contexts, users, and clusters. By default, all kubeconfig resources will be
named `${name}.${region}.${clusterName}`. For example, if the `Dev` account within the config file above had a cluster
within the `us-east-1` region named `test-v1.20`, the kubeconfig context would be named `Dev.us-east-1.test-v1.20`.
- `extraUsers` - Additional profiles to use when creating the kubeconfig contexts. This can be helpful when there are
multiple kubernetes users/groups setup within the cluster with their own permissions.

## Syncing Clusters

Running `gogok8s sync [accounts]` will look for EKS clusters in each account (and region) and fetch the necessary 
details to craft a kubeconfig cluster/user/context for connecting to the cluster. When no accounts are passed, all
accounts in the config file will be searched. This command supports the following flags:

- `--dry-run` - Performs a dryrun, only showing the kubeconfig diffs.
- `--purge` - Purges the kubeconfig of EKS clusters that were not found. This is off by default.

The `sync` command optionally accepts the accounts you want synced. Using the example config, if you want to only sync 
the `Dev` & `Staging` accounts you can run:
```bash
gogok8s sync Dev Staging
```

## Editing the Format

The `format` field supports some customization as to how the kubeconfig clusters, users, and contexts are named. The
following variables can be included as part of the format:
- `${name}` - The account name
- `${region}` - The AWS region of the cluster
- `${clusterName}` - The EKS cluster name
- `${clusterArn}` - The EKS cluster arn

## Shell Completions
Gogok8s uses [Cobra](https://github.com/spf13/cobra) and comes bundled with completions for the following shells:
- `bash`
- `fish`
- `powershell`
- `zsh`

Check out the instructions for adding completions for your preferred shell by running the following command (replacing `$YOUR_SHELL` with one from the list above):

```bash
gogok8s completion $YOUR_SHELL --help
```
