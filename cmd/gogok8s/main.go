package main

import (
	"errors"
	"os"

	"github.com/BigPapaChas/gogok8s/internal/commands"
	"github.com/BigPapaChas/gogok8s/internal/terminal"
)

func main() {
	if err := commands.Execute(); err != nil {
		if errors.Is(err, terminal.ErrUserQuit) {
			os.Exit(130)
		}

		terminal.PrintError(err.Error())
		os.Exit(1)
	}
}
