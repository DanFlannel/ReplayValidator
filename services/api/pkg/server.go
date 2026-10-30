package pkg

import (
	"context"
	"net/http"

	"github.com/sid-technologies/replayvalidator/db/ent"
	"github.com/sid-technologies/replayvalidator/pkg/errors"
	api "github.com/sid-technologies/replayvalidator/services/api/openapi"
)

type ClientImpl struct {
	//nolint:containedctx // matches the other SID services
	ctx       context.Context
	entClient *ent.Client
}

func NewClient(ctx context.Context, entClient *ent.Client) *ClientImpl {
	return &ClientImpl{ctx: ctx, entClient: entClient}
}

func (c *ClientImpl) CreateServer(ctx context.Context) (http.Handler, error) {
	apis := CreateRestService(ctx, c.entClient)

	ogenServer, err := api.NewServer(apis)
	if err != nil {
		return nil, errors.Wrap(err, "creating api server")
	}

	return ogenServer, nil
}
