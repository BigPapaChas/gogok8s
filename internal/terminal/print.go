package terminal

import (
	"fmt"

	"github.com/pterm/pterm"
)

func PrintBulletedWarnings(errors []error) {
	var prettyErrors []pterm.BulletListItem
	for _, err := range errors {
		prettyErrors = append(prettyErrors, pterm.BulletListItem{
			Level: 0,
			Text:  pterm.Yellow(err),
		})
	}

	_ = pterm.DefaultBulletList.WithItems(prettyErrors).Render()
}

func EnableDebug() {
	pterm.EnableDebugMessages()
}

func PrintDebug(message string) {
	pterm.Debug.Println(message)
}

func TextYellow(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.Yellow(message))
}

func TextWarning(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.Yellow(message, " ⚠"))
}

func TextSuccess(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.Green(message, " ✓"))
}

func PrintSuccess(message string) {
	pterm.Success.Printfln("%s", message)
}

func PrintWarning(message string) {
	pterm.Warning.Printfln("%s", message)
}

func PrintError(message string) {
	pterm.Error.Printfln("%s", message)
}

func StartNewSpinner(message string) (*pterm.SpinnerPrinter, error) {
	spinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start(message)
	if err != nil {
		return nil, fmt.Errorf("failed to start new spinner: %w", err)
	}

	return spinner, nil
}

func DiffAdd(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.LightGreen("+ ", message))
}

func DiffMinus(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.LightRed("- ", message))
}

func DiffModify(message string) {
	pterm.DefaultBasicText.Printfln("%s", pterm.LightYellow("~ ", message))
}
