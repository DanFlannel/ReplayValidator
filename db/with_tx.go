package db

import (
	"context"

	"github.com/sid-technologies/replayvalidator/db/ent"
	"github.com/sid-technologies/replayvalidator/pkg/errors"

	"github.com/rs/zerolog/log"
)

func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			err := tx.Rollback()
			if err != nil {
				log.Error().Msg("rolling back transaction")
			}

			panic(v)
		}
	}()

	err = fn(tx)
	if err != nil {
		rerr := tx.Rollback()
		if rerr != nil {
			err = errors.Wrap(err, "rolling back transaction")
		}

		return err
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "committing transaction")
	}

	return nil
}
