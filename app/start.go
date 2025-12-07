package app

import (
	"github.com/spf13/cobra"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/daemon"
)

func init() { //nolint: gochecknoinits
	startCmd.Flags().BoolVar(&devMode, "dev", false, "Enable dev mode")

	startCmd.Flags().BoolVar(
		&browseStatic,
		"browse",
		false,
		"Enable static file browsing (for development purposes only)",
	)

	rootCmd.AddCommand(startCmd)
}

var (
	configPath string // Path to the configuration file

	err          error
	devMode      bool
	browseStatic bool

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the GoPowerDNS-Admin web service",
		PreRun: func(_ *cobra.Command, _ []string) {
			if cfg, err = config.ReadConfig(configPath); err != nil {
				panic(err)
			}

			if devMode {
				cfg.DevMode = true
			}
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			daemon := daemon.New(&cfg)
			if err := daemon.Start(); err != nil {
				return err
			}

			return nil
		},
	}
)
