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

		return syncKubernetesClusters(dryRun, purge, args)
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

func syncKubernetesClusters(dryRun, purge bool, accounts []string) error {
	kubeconfig, err := kubecfg.LoadDefault()
	if err != nil {
		return fmt.Errorf("error reading from kubeconfig: %w", err)
	}

	var eksAccounts []clusters.EKSAccount

	if len(accounts) == 0 {
		// If no accounts were passed to the command, fetch from all accounts
		eksAccounts = cfg.Accounts
	} else {
		// If accounts were provided, only fetch from those accounts
		eksAccounts = cfg.ListAccountsFiltered(accounts)
	}

	if len(eksAccounts) == 0 {
		return nil
	}

	patch := clusters.GetPatchFromAccounts(eksAccounts)

	kubecfg.ApplyPatch(patch.Patch, kubeconfig, purge)

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
