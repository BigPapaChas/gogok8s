package commands

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/BigPapaChas/gogok8s/internal/config"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

//nolint:gochecknoglobals
var configCmd = &cobra.Command{
	Use:           "configure",
	Short:         "create a new account entry within the .gogok8s.yaml file",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			terminal.EnableDebug()
		}

		if cfg == nil {
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("error validating config: %w", err)
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
				return fmt.Errorf("failed to find user home directory: %w", err)
			}

			var defaultFilename string
			if cfgFile == "" {
				defaultFilename = path.Join(home, ".gogok8s.yaml")
			} else {
				defaultFilename = cfgFile
			}
			filename, err = terminal.PromptDefault("Gogok8s config file", defaultFilename)
			if err != nil {
				return fmt.Errorf("failed to get gogok8s config file: %w", err)
			}

			cfg = config.NewConfig()
		}

		accountName, err := terminal.PromptWithValidate("Account name", "", cfg.IsValidAccountName)
		if err != nil {
			return fmt.Errorf("failed to select AWS account: %w", err)
		}

		profile, err := terminal.PromptDefault("AWS Profile", "")
		if err != nil {
			return fmt.Errorf("failed to select AWS profile: %w", err)
		}

		regions, err := terminal.MultiSelect("AWS regions", config.ValidRegions)
		if err != nil {
			return fmt.Errorf("failed to select AWS regions: %w", err)
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
				return fmt.Errorf("failed to write %s config: %w", filename, err)
			}
		} else {
			// An existing configuration is being modified, write to the location that was used when running the command
			if err := cfg.Write(); err != nil {
				return fmt.Errorf("failed to write %s config: %w", viper.ConfigFileUsed(), err)
			}
		}
		terminal.TextSuccess(fmt.Sprintf("Account %s configured", accountName))

		return nil
	},
}
