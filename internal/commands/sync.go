package commands

import (
	"errors"
	"fmt"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/BigPapaChas/gogok8s/internal/kubecfg"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
	"github.com/spf13/cobra"
)

var errConfigNotExist = errors.New("couldn't find .gogok8s.yaml in home directory, try running `gogok8s configure`")

var syncCommand = &cobra.Command{
	Use:   "sync",
	Short: "syncs your kubeconfig with all available k8s clusters",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			terminal.EnableDebug()
		}
		if cfg == nil {
			return errConfigNotExist
		}

		return cfg.Validate()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		return syncKubernetesClusters(dryRun)
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func syncKubernetesClusters(dryRun bool) error {
	kubeconfig, err := kubecfg.LoadDefault()
	if err != nil {
		return fmt.Errorf("error reading from kubeconfig: %w", err)
	}

	patch := clusters.GetPatchFromAccounts(cfg.Accounts)

	kubecfg.ApplyPatch(patch.Patch, kubeconfig)

	if dryRun {
		terminal.TextSuccess("Dryrun complete")

		return nil
	}

	err = kubecfg.Write(kubeconfig)
	if err != nil {
		return err
	}

	terminal.TextSuccess("kubeconfig updated")

	return nil
}
