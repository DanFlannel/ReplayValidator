// Package cmd provides the cli for running the ReplayValidator API.
package cmd

import (
	libcmd "github.com/sid-technologies/replayvalidator/pkg/cmd"
	"github.com/sid-technologies/replayvalidator/services/api/app"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// New returns a new root cobra command that runs the API service.
func New() *cobra.Command {
	cmd := libcmd.NewRootCmd(
		"API",
		"ReplayValidator API",
	)

	cfg := app.DefaultConfig()
	bindFlags(cmd.Flags(), &cfg)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return app.Run(cmd.Context(), cfg)
	}

	return cmd
}

// bindFlags registers EVERY config field. viper's AutomaticEnv maps env vars
// onto registered flags only, so an unbound field cannot be configured at
// deploy time.
func bindFlags(flags *pflag.FlagSet, cfg *app.Config) {
	flags.StringVar(&cfg.ListenAddr, "listen-addr", cfg.ListenAddr, "Address to listen on")
	flags.StringVar(&cfg.DatabaseDBConn, "database-db-conn", cfg.DatabaseDBConn, "Database connection string (postgres:// or sqlite:)")
	flags.StringVar(&cfg.MonitoringAddr, "monitoring-addr", cfg.MonitoringAddr, "Address for monitoring/metrics endpoint")
	flags.StringVar(&cfg.OpenAPIPath, "openapi-path", cfg.OpenAPIPath, "Path the OpenAPI docs are served at")
	flags.StringVar(&cfg.APIVersion, "api-version", cfg.APIVersion, "API version segment")
	flags.StringVar(&cfg.AllowedOrigins, "allowed-origins", cfg.AllowedOrigins, "Comma-separated list of CORS allowed origins")
}
