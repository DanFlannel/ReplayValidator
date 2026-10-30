package log

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Cloud Logging reads structured stdout as JSON, but only if the fields are
// named the way it expects. zerolog's defaults are close but not the same, and
// the mismatch is silent: entries still appear, they just arrive with no
// severity at all.
const (
	// severityField is what Cloud Logging reads the level from.
	severityField = "severity"

	// errorEventType makes Cloud Error Reporting ingest an entry even without a
	// parsed stack trace. Without it, grouping depends on finding a stack trace
	// in the message, and Go errors logged as plain strings have none — so
	// errors would be visible in Logging and absent from Error Reporting.
	errorEventType = "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"
)

// InitializeLogger configures zerolog for Cloud Logging and attaches the
// Error Reporting hook to the global logger.
//
// Service and version come from Cloud Run's K_SERVICE / K_REVISION, so nothing
// needs passing in and local runs simply report as unknown.
func InitializeLogger() {
	// RFC3339, not Unix seconds: Cloud Logging parses `time` as a timestamp only
	// in RFC3339. A Unix integer is kept as an ordinary field and the entry is
	// stamped with ingestion time instead, which quietly reorders logs.
	zerolog.TimeFieldFormat = time.RFC3339Nano

	zerolog.LevelFieldName = severityField
	zerolog.MessageFieldName = "message"

	// Cloud Logging wants uppercase severity names, and its vocabulary is not
	// quite zerolog's: warn -> WARNING, fatal/panic -> CRITICAL.
	zerolog.LevelFieldMarshalFunc = cloudSeverity

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.LevelColors = map[zerolog.Level]int{
		zerolog.DebugLevel: 36,
		zerolog.InfoLevel:  34,
		zerolog.WarnLevel:  33,
		zerolog.ErrorLevel: 31,
	}

	// Attached to the global logger rather than at each call site: a reporting
	// path that depends on every author remembering a field is one that silently
	// stops covering new code.
	log.Logger = log.Logger.Hook(NewErrorReportingHook("", ""))
}

// cloudSeverity maps a zerolog level to the Cloud Logging severity vocabulary.
func cloudSeverity(l zerolog.Level) string {
	switch l {
	case zerolog.DebugLevel:
		return "DEBUG"
	case zerolog.InfoLevel:
		return "INFO"
	case zerolog.WarnLevel:
		return "WARNING"
	case zerolog.ErrorLevel:
		return "ERROR"
	case zerolog.FatalLevel, zerolog.PanicLevel:
		return "CRITICAL"
	default:
		// NoLevel, Disabled, Trace and anything zerolog adds later. DEFAULT is
		// Cloud Logging's "unspecified", which is the honest answer for a level
		// this mapping does not know.
		return "DEFAULT"
	}
}

// ErrorReportingHook tags error-and-above entries so Cloud Error Reporting
// ingests them.
type ErrorReportingHook struct {
	service string
	version string
}

// NewErrorReportingHook builds the hook for a service. Version is best-effort:
// an unknown build still reports, it just cannot be attributed to a release.
func NewErrorReportingHook(service, version string) ErrorReportingHook {
	if service == "" {
		service = strings.TrimSpace(os.Getenv("K_SERVICE"))
	}

	if version == "" {
		version = strings.TrimSpace(os.Getenv("K_REVISION"))
	}

	return ErrorReportingHook{service: service, version: version}
}

// Run implements zerolog.Hook.
func (h ErrorReportingHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	if level < zerolog.ErrorLevel {
		return
	}

	e.Str("@type", errorEventType)

	// serviceContext is what Error Reporting groups by. Without it every service
	// lands in one undifferentiated bucket.
	e.Dict("serviceContext", zerolog.Dict().
		Str("service", h.service).
		Str("version", h.version))
}
