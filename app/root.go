// Package app implements the main application commands.
package app

import (
	"github.com/spf13/cobra"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version"
)

var (
	dumpConfig *bool
	cfg        config.Config

	rootCmd = &cobra.Command{
		Use:     "go-powerdns-admin",
		Version: version.Version,
		Short:   "GoPowerDNS-Admin is a web-based management tool for PowerDNS",
		Long: `GoPowerDNS-Admin is a web-based management tool for PowerDNS
that provides an easy-to-use interface for managing domains, records, and users.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Handle --dump-config flag
			if dumpConfig != nil && *dumpConfig {
				if cfg, err = config.ReadConfig(configPath); err != nil {
					return err
				}

				var out string

				out, err = config.DumpConfigJSON(&cfg)
				if err != nil {
					return err
				}

				cmd.Println(out)
				return nil
			}

			// Otherwise show help when no subcommand is provided
			return cmd.Help()
		},
	}
)

// Execute runs the root command.
func Execute() error {
	dumpConfig = rootCmd.PersistentFlags().Bool("dump-config", false, "dump effective config as JSON")

	rootCmd.PersistentFlags().StringVarP(
		&configPath,
		"configPath",
		"c",
		"etc/",
		"Path to the configuration folder",
	)

	return rootCmd.Execute()
}
