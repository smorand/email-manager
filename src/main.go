package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "email-manager",
	Short: "Gmail Manager - Manage Gmail emails",
	Long:  "Send, receive, search, and manage Gmail emails using Gmail API v1",
}

func main() {
	// Setup command flags
	setupSendFlags()
	setupListFlags()
	setupSearchFlags()
	setupDownloadAttachmentsFlags()
	setupLabelCommands()

	// Register all commands
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(unreadCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(downloadAttachmentsCmd)
	rootCmd.AddCommand(labelsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
