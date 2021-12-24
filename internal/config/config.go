package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
)

type Config struct {
	Accounts []clusters.EKSAccount `yaml:"accounts"`
}

var (
	ValidRegions = []string{
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
	validRegionsMap = getValidRegions()
)

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	// TODO: Add more validation

	// Validate accounts
	for _, account := range c.Accounts {
		for _, region := range account.Regions {
			if !isValidRegion(region) {
				return fmt.Errorf("invalid region %s in account %s", region, account.Name)
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
		return err
	}

	return ioutil.WriteFile(filename, data, os.FileMode(0o644))
}

func (c *Config) AddAccount(account clusters.EKSAccount) {
	c.Accounts = append(c.Accounts, account)
}

func isValidRegion(region string) bool {
	_, ok := validRegionsMap[region]
	return ok
}

func getValidRegions() map[string]struct{} {
	regions := make(map[string]struct{})
	for _, region := range ValidRegions {
		regions[region] = struct{}{}
	}
	return regions
}
