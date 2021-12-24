package commands

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/BigPapaChas/gogok8s/internal/config"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

var configCmd = &cobra.Command{
	Use:           "configure",
	Short:         "configure the .gogok8s.yaml file",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			terminal.EnableDebug()
		}
		if cfg != nil {
			return cfg.Validate()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var filename string
		var err error
		if cfg == nil {
			// An existing gogok8s config file was not found, prompt user for filename to use
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			var defaultFilename string
			if cfgFile == "" {
				defaultFilename = path.Join(home, ".gogok8s.yaml")
			} else {
				defaultFilename = cfgFile
			}
			filename, err = terminal.Prompt("Gogok8s config file", defaultFilename)
			if err != nil {
				return err
			}

			cfg = config.NewConfig()
		}

		accountName, err := terminal.Prompt("Account name", "")
		if err != nil {
			return err
		}

		profile, err := terminal.Prompt("AWS Profile", "")
		if err != nil {
			return err
		}

		regions, err := terminal.Select("AWS regions", config.ValidRegions)
		if err != nil {
			return err
		}

		account := clusters.EKSAccount{
			Profile: profile,
			Regions: regions,
			Name:    accountName,
		}
		cfg.AddAccount(account)

		if filename != "" {
			// A new configuration file was created, write the updated config to the user-specified filename
			terminal.PrintDebug(filename)
			if err := cfg.WriteToFile(filename); err != nil {
				return err
			}
		} else {
			// An existing configuration is being modified, write to the location that was used when running the command
			if err := cfg.Write(); err != nil {
				return err
			}
		}
		terminal.TextSuccess(fmt.Sprintf("account %s added", accountName))
		return nil
	},
}
