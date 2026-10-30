// Package apierr is the single place a Go error becomes an HTTP status.
package apierr

import (
	"context"

	"github.com/sid-technologies/replayvalidator/db/ent"
	"github.com/sid-technologies/replayvalidator/pkg/errors"

	"github.com/rs/zerolog/log"
)

// Response is the classified form of an error: what leaves the API.
type Response struct {
	Status  int
	Code    string
	Message string
}

// Classify decides how an error leaves the API.
//
// Kinds first — classification decided where the error was created. Then the
// ent's well-known error shapes. Anything unclassified is a server fault:
// logged here once WITH its structured attrs, returned sanitized — err.Error()
// on a 500 leaks connection strings and SQL.
func Classify(_ context.Context, err error) Response {
	switch errors.KindOf(err) {
	case errors.KindInvalid:
		return Response{Status: 400, Code: "bad_request", Message: err.Error()}
	case errors.KindNotFound:
		return Response{Status: 404, Code: "not_found", Message: err.Error()}
	case errors.KindConflict:
		return Response{Status: 409, Code: "conflict", Message: err.Error()}
	case errors.KindForbidden:
		return Response{Status: 403, Code: "forbidden", Message: err.Error()}
	case errors.KindUnauthorized:
		return Response{Status: 401, Code: "unauthorized", Message: err.Error()}
	case errors.KindPlanRequired:
		// The cap's own message, not the wrap chain above it.
		msg, _ := errors.PlanLimitMessage(err)

		return Response{Status: 402, Code: "plan_required", Message: msg}
	default:
	}

	switch {
	case ent.IsNotFound(err):
		return Response{Status: 404, Code: "not_found", Message: err.Error()}

	case ent.IsConstraintError(err):
		return Response{Status: 409, Code: "conflict", Message: err.Error()}
	}

	log.Error().Err(err).Fields(errors.AttrsOf(err)).Msg("unhandled API error")

	return Response{Status: 500, Code: "internal_error", Message: "internal server error"}
}
