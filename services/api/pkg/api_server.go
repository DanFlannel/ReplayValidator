package pkg

import (
	"context"

	"github.com/sid-technologies/replayvalidator/db"
	"github.com/sid-technologies/replayvalidator/db/ent"
	"github.com/sid-technologies/replayvalidator/pkg/apierr"
	"github.com/sid-technologies/replayvalidator/pkg/buildinfo"
	api "github.com/sid-technologies/replayvalidator/services/api/openapi"
)

// RestService implements the ogen api.Handler.
//
// It deliberately does NOT embed api.UnimplementedHandler. Embedding it makes
// a forgotten operation compile and answer 501 at runtime instead of failing
// the build.
type RestService struct {
	//nolint:containedctx // matches the other SID services
	ctx       context.Context
	entClient *ent.Client
}

func CreateRestService(ctx context.Context, entClient *ent.Client) *RestService {
	return &RestService{ctx: ctx, entClient: entClient}
}

// HealthCheck implements api.Handler.
//
// Deliberately shallow — it answers whether the process is up, not whether
// its dependencies are. See ReadyCheck for that.
func (r *RestService) HealthCheck(_ context.Context) (*api.HealthCheckResponse, error) {
	return &api.HealthCheckResponse{
		Status:  api.HealthCheckResponseStatusOk,
		Version: api.NewOptString(buildinfo.Version()),
	}, nil
}

// ReadyCheck implements api.Handler.
func (r *RestService) ReadyCheck(ctx context.Context) (api.ReadyCheckRes, error) {
	checks := api.ReadyCheckResponseChecks{}

	ready := true

	err := db.Ping(ctx, r.entClient)
	if err != nil {
		ready = false
		checks["database"] = err.Error()
	} else {
		checks["database"] = "ok"
	}

	resp := api.ReadyCheckResponse{
		Ready:  ready,
		Checks: api.NewOptReadyCheckResponseChecks(checks),
	}

	if !ready {
		out := api.ReadyCheckServiceUnavailable(resp)
		return &out, nil
	}

	out := api.ReadyCheckOK(resp)
	return &out, nil
}

// NewError classifies rather than leaks.
//
// The generator's default returns 501 with err.Error() verbatim, which is
// both the wrong status and an internals leak. apierr.Classify maps kinded
// errors to real statuses and sanitizes anything unrecognized to a plain 500.
func (*RestService) NewError(_ context.Context, err error) *api.UnexpectedStatusCode {
	resp := apierr.Classify(context.Background(), err)

	return &api.UnexpectedStatusCode{
		StatusCode: resp.Status,
		Response: api.Error{
			Code:    int32(resp.Status), //nolint:gosec // HTTP status codes fit an int32
			Message: resp.Message,
		},
	}
}
