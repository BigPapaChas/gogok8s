package clusters

import (
	"gogok8s/internal/kubecfg"
	"gogok8s/internal/terminal"
)

type PatchResult struct {
	Patch       *kubecfg.KubeConfigPatch
	Errors      []error
	AccountName string
}

func GetPatchFromAccounts(accounts []EKSAccount) PatchResult {
	patch := PatchResult{
		Patch: &kubecfg.KubeConfigPatch{},
	}
	c := make(chan PatchResult, len(accounts))

	spinner, _ := terminal.StartNewSpinner("Fetching EKS clusters from accounts...")
	for _, account := range accounts {
		go func(account EKSAccount) {
			accountPatch, accountErrors := account.GenerateKubeConfigPatch()

			c <- PatchResult{
				Patch:       accountPatch,
				Errors:      accountErrors,
				AccountName: account.Name,
			}
		}(account)
	}

	for range accounts {
		result := <-c
		patch.Patch.Users = append(patch.Patch.Users, result.Patch.Users...)
		patch.Patch.Clusters = append(patch.Patch.Clusters, result.Patch.Clusters...)
		patch.Patch.Contexts = append(patch.Patch.Contexts, result.Patch.Contexts...)
		patch.Errors = append(patch.Errors, result.Errors...)
		if len(result.Errors) > 0 {
			terminal.TextWarning(result.AccountName)
			terminal.PrintBulletedWarnings(result.Errors)
		} else {
			terminal.TextSuccess(result.AccountName)
		}
	}
	_ = spinner.Stop()

	return patch
}
