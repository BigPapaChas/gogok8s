package main

import (
	"os"

	"github.com/BigPapaChas/gogok8s/internal/commands"
)

func main() {
	os.Exit(commands.Execute())
}
