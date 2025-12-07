// Package app implements the main application commands.
package app

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-powerdns-admin",
	Short: "GoPowerDNS-Admin is a web-based management tool for PowerDNS",
	Long: `GoPowerDNS-Admin is a web-based management tool for PowerDNS
that provides an easy-to-use interface for managing domains, records, and users.`,
	Args: cobra.OnlyValidArgs,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
