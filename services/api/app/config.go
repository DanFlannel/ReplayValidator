package app

import (
	"github.com/sid-technologies/replayvalidator/pkg/buildinfo"
)

const (
	defaultListenAddr     = ":8080"
	defaultDatabaseDBConn = "" // Must be set via environment variable
	// Shared monitoring port convention used across SID services.
	defaultMonitoringAddr = ":26662"
	defaultOpenAPIPath    = "/openapi"
	defaultAPIVersion     = "v1"
)

// Config for the ReplayValidator API.
//
// Every field here MUST have a flag bound in cmd.bindFlags. Config reaches the
// process through viper's AutomaticEnv, which only maps environment variables
// onto registered flags — a field with no flag is silently unreachable from the
// environment no matter what the deploy sets.
type Config struct {
	ListenAddr     string
	DatabaseDBConn string
	MonitoringAddr string
	OpenAPIPath    string
	APIVersion     string
	Version        string
	AllowedOrigins string
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:     defaultListenAddr,
		DatabaseDBConn: defaultDatabaseDBConn,
		MonitoringAddr: defaultMonitoringAddr,
		OpenAPIPath:    defaultOpenAPIPath,
		APIVersion:     defaultAPIVersion,
		Version:        buildinfo.Version(),
	}
}
