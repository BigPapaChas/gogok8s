package commands

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BigPapaChas/gogok8s/internal/config"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

const (
	exitCodeUserQuit = 130
	exitCodeError    = 1
	exitCodeNoError  = 0
)

var (
	cfgFile string         //nolint:gochecknoglobals
	cfg     *config.Config //nolint:gochecknoglobals
	debug   bool           //nolint:gochecknoglobals
)

//nolint:gochecknoglobals
var rootCmd = &cobra.Command{
	Use:     "gogok8s",
	Short:   "gogok8s helps manage your k8s cluster kubeconfig(s)",
	Version: "v1.2.0",
}

//nolint:gochecknoinits
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gogok8s.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug messages")

	syncCommand.Flags().Bool("dry-run", false, "performs a dryrun, showing a diff of the changes")
	syncCommand.Flags().Bool("purge", false, "purges the kubeconfig of clusters not found")
	rootCmd.AddCommand(syncCommand)

	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gogok8s")
	}

	if err := viper.ReadInConfig(); err != nil {
		terminal.PrintWarning(err.Error())
	} else {
		cfg = config.NewConfig()
		cobra.CheckErr(viper.Unmarshal(cfg))
	}
}

func Execute() int {
	var code int

	if err := rootCmd.Execute(); err != nil {
		if errors.Is(errors.Unwrap(err), terminal.ErrUserQuit) {
			code = exitCodeUserQuit
		} else {
			terminal.PrintError(err.Error())
			code = exitCodeError
		}
	} else {
		code = exitCodeNoError
	}

	return code
}
