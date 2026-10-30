// Package app runs the ReplayValidator API.
//
// This is a bootstrap scaffold: health/ready only, no auth, no domain logic.
// It exists to prove the BE+FE wiring works end to end so the real domain can
// be built out on top of it.
package app

import (
	"context"
	"net/http"
	"strings"

	"github.com/sid-technologies/replayvalidator/db"
	"github.com/sid-technologies/replayvalidator/db/ent"
	api "github.com/sid-technologies/replayvalidator/pkg/api_server"
	"github.com/sid-technologies/replayvalidator/pkg/buildinfo"
	logger "github.com/sid-technologies/replayvalidator/pkg/log"
	"github.com/sid-technologies/replayvalidator/services/api/pkg"

	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context, cfg Config) error {
	logger.InitializeLogger()

	log.Info().Msg("Starting API Service")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	buildinfo.Instrument(ctx)
	log.Info().Msgf("Version info | version: %s", buildinfo.Version())

	entCl, err := db.CreateDBClient(ctx, cfg.DatabaseDBConn)
	if err != nil {
		return err
	}
	defer db.CloseDBClient(entCl)

	err = db.CreateSchema(ctx, entCl)
	if err != nil {
		return err
	}

	restHandler := CreateRESTClient(ctx, entCl)

	var origins []string
	if cfg.AllowedOrigins != "" {
		origins = strings.Split(cfg.AllowedOrigins, ",")
	}

	corsCfg := api.CORSConfig{AllowedOrigins: origins}
	httpServer := api.CreateHTTPClientWithConfig(restHandler, cfg.ListenAddr, cfg.APIVersion, cfg.OpenAPIPath,
		corsCfg, api.InstrumentHandler("api service"))

	go func() {
		monErr := api.ServeMonitoring(ctx, ctx.Done(), cfg.MonitoringAddr)
		if monErr != nil {
			log.Warn().Err(monErr).Msg("monitoring server stopped")
		}
	}()

	return api.StartServer(ctx, httpServer, cancel)
}

// CreateRESTClient builds the ogen handler.
func CreateRESTClient(ctx context.Context, entCl *ent.Client) http.Handler {
	client := pkg.NewClient(ctx, entCl)

	handler, err := client.CreateServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create api server")
	}

	return handler
}
