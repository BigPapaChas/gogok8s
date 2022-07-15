package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

type Config struct {
	Accounts []clusters.EKSAccount `yaml:"accounts"`
}

//nolint:gochecknoglobals
var ValidRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"ca-central-1",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"sa-east-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-northeast-3",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-south-1",
}

const configFilemode = os.FileMode(0o644)

var (
	ErrDuplicateAccountName = errors.New("Account name already exists")
	ErrInvalidAWSRegion     = errors.New("invalid AWS region")
	ErrMustContainAWSRegion = errors.New("account must contain at least one region")
)

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	accountNames := make(map[string]struct{})

	for idx, account := range c.Accounts {
		// validate that there are no duplicate account names
		if _, ok := accountNames[account.Name]; !ok {
			accountNames[account.Name] = struct{}{}
		} else {
			return duplicateAccountError(fmt.Sprintf("`%s` at accounts[%d]", account.Name, idx))
		}

		// validate that each account has at least one valid region
		if len(account.Regions) == 0 {
			return accountHasNoRegionsError(account.Name)
		}

		// validate each region is a valid AWS region
		for _, region := range account.Regions {
			if !isValidRegion(region) {
				return invalidRegionError(fmt.Sprintf("region %s in account %s", region, account.Name))
			}
		}
	}

	return nil
}

func (c *Config) Write() error {
	return c.WriteToFile(viper.ConfigFileUsed())
}

func (c *Config) WriteToFile(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config yaml: %w", err)
	}

	if err = os.WriteFile(filename, data, configFilemode); err != nil {
		return fmt.Errorf("failed to write config yaml to file: %w", err)
	}

	return nil
}

func (c *Config) AddAccount(account clusters.EKSAccount) {
	c.Accounts = append(c.Accounts, account)
}

func (c *Config) ListAccountsFiltered(filter []string) []clusters.EKSAccount {
	filterAccounts := make(map[string]struct{})
	for _, account := range filter {
		filterAccounts[account] = struct{}{}
	}

	var accounts []clusters.EKSAccount

	for _, account := range c.Accounts {
		if _, ok := filterAccounts[account.Name]; ok {
			accounts = append(accounts, account)
			delete(filterAccounts, account.Name)
		}
	}

	for account := range filterAccounts {
		terminal.PrintWarning(fmt.Sprintf("can't find account `%s`", account))
	}

	return accounts
}

func (c *Config) ListAccountNamesFiltered(excludeFilter []string) []string {
	excludeAccounts := make(map[string]struct{})
	for _, account := range excludeFilter {
		excludeAccounts[account] = struct{}{}
	}

	var accountNames []string

	for _, account := range c.Accounts {
		if _, ok := excludeAccounts[account.Name]; !ok {
			accountNames = append(accountNames, account.Name)
		}
	}

	return accountNames
}

func (c *Config) IsValidAccountName(name string) error {
	for _, account := range c.Accounts {
		if name == account.Name {
			return duplicateAccountError(name)
		}
	}

	return nil
}

func duplicateAccountError(msg string) error {
	return fmt.Errorf("%w: %s", ErrDuplicateAccountName, msg)
}

func invalidRegionError(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidAWSRegion, msg)
}

func accountHasNoRegionsError(msg string) error {
	return fmt.Errorf("%w: %s", ErrMustContainAWSRegion, msg)
}

func isValidRegion(region string) bool {
	for _, validRegion := range ValidRegions {
		if region == validRegion {
			return true
		}
	}

	return false
}
