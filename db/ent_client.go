package db

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/rs/zerolog/log"
	"github.com/sid-technologies/replayvalidator/db/ent"
	"github.com/sid-technologies/replayvalidator/pkg/errors"

	_ "github.com/jackc/pgx/v5/stdlib"                            // importing postgres sql driver
	_ "github.com/mattn/go-sqlite3"                                // importing sqlite driver
	_ "github.com/sid-technologies/replayvalidator/db/ent/runtime" // import the runtime package to register hooks
)

func NewPostgressClient(dbURL string) (*ent.Client, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open postgres connection")
	}

	// Create an ent.Driver from `db`.
	drv := entsql.OpenDB(dialect.Postgres, db)

	client := ent.NewClient(ent.Driver(drv))

	// ent.Client exposes no way to reach its own pool, and readiness needs a
	// real round trip rather than a nil check — a pooled handle exists whether
	// or not the server behind it is up.
	pools.Store(client, db)

	return client, nil
}

func NewSQLiteClient() (*ent.Client, error) {
	client, err := ent.Open(dialect.SQLite, "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		return nil, errors.Wrap(err, "failed to open sqlite connection")
	}

	return client, nil
}

// CreateSchema creates all schema resources and runs all required migrations.
func CreateSchema(ctx context.Context, client *ent.Client) error {
	err := client.Schema.Create(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create schema for db")
	}

	return nil
}

// CreateDBClient opens a database client from a connection string.
//
// A `file:` or `sqlite:` DSN opens SQLite; anything else is treated as Postgres.
// This exists so local development can run against a throwaway database
// instead of a bypass flag.
//
// Production connection strings are postgres:// URLs, so this cannot silently
// downgrade a real deployment.
func CreateDBClient(ctx context.Context, dbURL string) (*ent.Client, error) {
	if isSQLiteDSN(dbURL) {
		entCl, err := newSQLiteClientFromDSN(ctx, dbURL)
		if err != nil {
			return nil, errors.Wrap(err, "create sqlite db client")
		}

		return entCl, nil
	}

	entCl, err := NewPostgressClient(dbURL)
	if err != nil {
		return nil, errors.Wrap(err, "create db client")
	}

	return entCl, nil
}

// isSQLiteDSN reports whether a connection string names a SQLite database.
func isSQLiteDSN(dbURL string) bool {
	return strings.HasPrefix(dbURL, "file:") || strings.HasPrefix(dbURL, "sqlite:")
}

// newSQLiteClientFromDSN opens SQLite at the given DSN and creates the schema.
//
// The schema is created here because a SQLite database is always fresh — there
// is no migrations job for a throwaway file, and an empty database would fail
// every query with "no such table" rather than anything explanatory.
func newSQLiteClientFromDSN(ctx context.Context, dbURL string) (*ent.Client, error) {
	dsn := strings.TrimPrefix(dbURL, "sqlite:")

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open sqlite connection")
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	pools.Store(client, db)

	err = client.Schema.Create(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "creating sqlite schema")
	}

	return client, nil
}

func CloseDBClient(entCl *ent.Client) {
	err := entCl.Close()
	if err != nil {
		log.Error().Msgf("Failed to close ent client %v", err)
	}
}

// pools maps each ent client to the database handle behind it, so readiness can
// issue a real query. ent keeps its driver unexported.
var pools sync.Map

// Ping verifies the database is reachable and answering queries.
//
// It issues a real round trip rather than inspecting the pool: a pooled handle
// exists whether or not the server behind it is up, so checking for a non-nil
// client proves only that Open() was called.
func Ping(ctx context.Context, entCl *ent.Client) error {
	if entCl == nil {
		return errors.New("no database client")
	}

	handle, ok := pools.Load(entCl)
	if !ok {
		return errors.New("database handle not registered for this client")
	}

	pool, ok := handle.(*sql.DB)
	if !ok {
		return errors.New("registered database handle has the wrong type")
	}

	err := pool.PingContext(ctx)
	if err != nil {
		return errors.Wrap(err, "database unreachable")
	}

	return nil
}
