package config

import (
	"fmt"

	"github.com/BigPapaChas/gogok8s/internal/clusters"
)

type Config struct {
	Accounts []clusters.EKSAccount `yaml:"accounts"`
}

var (
	validRegions = []string{
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

func isValidRegion(region string) bool {
	_, ok := validRegionsMap[region]
	return ok
}

func getValidRegions() map[string]struct{} {
	regions := make(map[string]struct{})
	for _, region := range validRegions {
		regions[region] = struct{}{}
	}
	return regions
}
