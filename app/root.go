// Package app implements the main application commands.
package app

import (
	"github.com/spf13/cobra"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

var (
	dumpConfig *bool
	jsonOutput *bool
	cfg        config.Config

	rootCmd = &cobra.Command{
		Use:   "go-powerdns-admin",
		Short: "GoPowerDNS-Admin is a web-based management tool for PowerDNS",
		Long: `GoPowerDNS-Admin is a web-based management tool for PowerDNS
that provides an easy-to-use interface for managing domains, records, and users.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Handle --dump-config flag
			if dumpConfig != nil && *dumpConfig {
				var out string

				if cfg, err = config.ReadConfig(configPath); err != nil {
					return err
				}

				if jsonOutput != nil && *jsonOutput {
					out, err = config.DumpConfigJSON(&cfg)
				} else {
					out, err = config.DumpConfig(&cfg)
				}

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
	dumpConfig = rootCmd.PersistentFlags().Bool("dump-config", false, "dump config file")
	jsonOutput = rootCmd.PersistentFlags().Bool("json", false, "output in JSON format")

	rootCmd.PersistentFlags().StringVarP(
		&configPath,
		"configPath",
		"c",
		"etc/",
		"Path to the configuration folder",
	)

	return rootCmd.Execute()
}
