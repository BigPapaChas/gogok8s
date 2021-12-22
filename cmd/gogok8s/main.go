package main

import (
	"os"

	"github.com/BigPapaChas/gogok8s/internal/commands"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

func main() {
	if err := commands.Execute(); err != nil {
		terminal.PrintError(err.Error())
		os.Exit(1)
	}
}
