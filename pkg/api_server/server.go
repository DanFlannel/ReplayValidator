// Package apiserver provides HTTP server utilities for ReplayValidator services.
package apiserver

import (
	"context"
	"net/http"
	"time"

	"github.com/sid-technologies/replayvalidator/pkg/docs"
	"github.com/sid-technologies/replayvalidator/pkg/errors"

	"github.com/justinas/alice"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// CORSConfig holds CORS configuration for API servers.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins. An EMPTY list allows
	// nothing — see CreateHTTPClientWithConfig, which has to say so explicitly
	// because rs/cors reads an empty list as "allow all".
	AllowedOrigins []string
}

// DefaultCORSConfig returns a configuration that allows no cross-origin request.
//
// This function and the field above both used to say "restrictive" while
// producing the opposite: rs/cors treats an empty AllowedOrigins as allow-all
// (cors.go:163, "Default is all origins"), and AllowCredentials was true
// alongside it.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{},
	}
}

// MaxRequestBodySize is the default maximum request body size (10MB).
const MaxRequestBodySize = 10 * 1024 * 1024

// CreateHTTPClient creates a new http.Server with API prefix and docs.
// This matches the sid pattern for services with OpenAPI docs.
func CreateHTTPClient(handler http.Handler, listenAddr string, apiVersion string, openAPIPath string, middlewares ...func(handler http.Handler) http.Handler) *http.Server {
	return CreateHTTPClientWithConfig(handler, listenAddr, apiVersion, openAPIPath, DefaultCORSConfig(), middlewares...)
}

// CreateHTTPClientWithConfig creates a new http.Server with CORS config and body size limits.
func CreateHTTPClientWithConfig(handler http.Handler, listenAddr string, apiVersion string, openAPIPath string, corsCfg CORSConfig, middlewares ...func(handler http.Handler) http.Handler) *http.Server {
	chain := alice.New()
	for _, m := range middlewares {
		chain = chain.Append(m)
	}

	log.Info().Msgf("API version: %s", apiVersion)

	// Create http server
	mux := http.NewServeMux()

	// Serve API at root (gateway doesn't need /api/v1 prefix)
	mux.Handle("/", handler)

	// Every service configures an OpenAPIPath and passes it here; it used to be
	// discarded, so the docs were never served anywhere. Registered before "/"
	// would swallow it — ServeMux prefers the longer pattern, but only if one
	// exists.
	if openAPIPath != "" {
		docsHandler := docs.GetHandler(openAPIPath)
		mux.Handle(openAPIPath, docsHandler)
		mux.Handle(openAPIPath+"/", docsHandler)
	}

	// Use configurable CORS origins instead of wildcard
	corsOpts := cors.Options{
		AllowedOrigins: corsCfg.AllowedOrigins,
		// PATCH is not optional: dashboards may send it for updates.
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Organization-Id", "X-Customer-Id", "X-Product-Slug", "X-CSRF-Token"},
		AllowCredentials: true,
		// Exports name their file here; the browser hides it cross-origin otherwise.
		ExposedHeaders: []string{"Content-Disposition"},
	}

	// An empty origins list must DENY. rs/cors reads an empty AllowedOrigins as
	// allow-all and then emits `*` rather than the echoed origin — which
	// browsers reject for credentialed requests, so the config meant to enable
	// the dashboard silently broke it while leaving every unauthenticated
	// endpoint world-readable cross-origin.
	//
	// AllowOriginFunc is checked BEFORE the empty-list branch in rs/cors, so it
	// is the only way to express "none" to that library.
	if len(corsCfg.AllowedOrigins) == 0 {
		log.Warn().Msg("no ALLOWED_ORIGINS configured — all cross-origin requests will be refused")

		corsOpts.AllowOriginFunc = func(string) bool { return false }
	}

	c := cors.New(corsOpts)
	corsHandler := c.Handler(chain.Then(mux))

	// Enforce request body size limits
	limitedHandler := http.MaxBytesHandler(corsHandler, MaxRequestBodySize)

	httpServer := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		Handler:           limitedHandler,
	}

	return httpServer
}

// StartServer starts the HTTP server and handles graceful shutdown.
func StartServer(ctx context.Context, httpServer *http.Server, cancel context.CancelFunc) error {
	var eg errgroup.Group

	eg.Go(func() error {
		defer cancel() // Cancel the app context if serving fails.

		log.Info().Str("addr", httpServer.Addr).Msg("Starting HTTP server")

		return errors.Wrap(httpServer.ListenAndServe(), "serve")
	})

	eg.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("Shutdown detected, stopping server")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		err := httpServer.Shutdown(shutdownCtx) //nolint:contextcheck // intentional new context for graceful shutdown
		if err != nil {
			log.Error().Err(err).Msg("Failed to shutdown server")
			return errors.Wrap(err, "server shutdown")
		}

		return nil
	})

	err := eg.Wait()
	if errors.Is(err, http.ErrServerClosed) {
		return nil // No error on shutdown.
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to start server")
		return errors.Wrap(err, "server")
	}

	return nil
}
