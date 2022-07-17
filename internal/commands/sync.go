package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/BigPapaChas/gogok8s/internal/kubecfg"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

var errConfigNotExist = errors.New("couldn't find .gogok8s.yaml in home directory, try running `gogok8s configure`")

//nolint:gochecknoglobals
var syncCommand = &cobra.Command{
	Use:   "sync [accounts]",
	Short: "syncs your kubeconfig with all available k8s clusters",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			terminal.EnableDebug()
		}
		if cfg == nil {
			return errConfigNotExist
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("error validating config: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		purge, _ := cmd.Flags().GetBool("purge")

		return syncKubernetesClusters(args, dryRun, purge)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if cfg != nil {
			return cfg.ListAccountNamesFiltered(args), cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func syncKubernetesClusters(accounts []string, dryRun, purge bool) error {
	kubeconfig, err := kubecfg.LoadDefault()
	if err != nil {
		return fmt.Errorf("error reading from kubeconfig: %w", err)
	}

	// If no accounts were passed to the command, fetch from all accounts
	var eksAccounts []clusters.ClusterAccount
	if len(accounts) == 0 {
		eksAccounts = cfg.GetAccounts()
	} else {
		eksAccounts = cfg.ListAccountsFiltered(accounts)
	}

	if len(eksAccounts) == 0 {
		return nil
	}

	patch := fetchKubeConfigFromAccounts(eksAccounts)

	kubecfg.ApplyPatch(patch, kubeconfig, purge)

	if dryRun {
		terminal.TextSuccess("Dryrun complete")

		return nil
	}

	err = kubecfg.Write(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to write to kubeconfig: %w", err)
	}

	terminal.TextSuccess("kubeconfig updated")

	return nil
}

type KubeConfigResult struct {
	Patch       *kubecfg.KubeConfigPatch
	Errors      []error
	AccountName string
}

func fetchKubeConfigFromAccounts(accounts []clusters.ClusterAccount) *kubecfg.KubeConfigPatch {
	kubeconfig := &kubecfg.KubeConfigPatch{}
	spinner, _ := terminal.StartNewSpinner("Scanning accounts for Kubernetes clusters...")
	ch := make(chan KubeConfigResult, len(accounts))

	for _, account := range accounts {
		go func(account clusters.ClusterAccount) {
			kubeconfigPatch, errors := account.GenerateKubeConfig()

			ch <- KubeConfigResult{
				Patch:       kubeconfigPatch,
				Errors:      errors,
				AccountName: account.PrettyName(),
			}
		}(account)
	}

	for range accounts {
		result := <-ch

		kubeconfig.Clusters = append(kubeconfig.Clusters, result.Patch.Clusters...)
		kubeconfig.Users = append(kubeconfig.Users, result.Patch.Users...)
		kubeconfig.Contexts = append(kubeconfig.Contexts, result.Patch.Contexts...)

		if len(result.Errors) > 0 {
			terminal.TextWarning(result.AccountName)
			terminal.PrintBulletedWarnings(result.Errors)
		} else {
			terminal.TextSuccess(result.AccountName)
		}
	}

	_ = spinner.Stop()

	return kubeconfig
}
