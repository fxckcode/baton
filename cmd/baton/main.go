package main

import (
	"os"

	"github.com/fxckcode/baton/internal/cli"
)

func main() {
	os.Exit(cli.Run(cli.Options{
		Args:   os.Args[1:],
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
