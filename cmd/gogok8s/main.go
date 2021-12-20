package main

import (
	"gogok8s/internal/commands"
	"gogok8s/internal/terminal"
	"os"
)

func main() {
	if err := commands.Execute(); err != nil {
		terminal.PrintError(err.Error())
		os.Exit(1)
	}
}
