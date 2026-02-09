package main

import (
	"fmt"
	"os"

	"github.com/dslh/zh/cmd"
	"github.com/dslh/zh/internal/exitcode"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(exitcode.ExitCode(err))
	}
}
