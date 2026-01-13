// Package main is the entry point for email-manager.
package main

import (
	"fmt"
	"os"

	"email-manager/internal/cli"
)

func main() {
	cli.Init()

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
