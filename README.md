# ReplayValidator

Replay Validation.

## Status

Bootstrap scaffold. This repo replicates the structural pattern shared by
SID's other Go+React products (`services/<name>` with ogen-generated
OpenAPI, `db/ent` for the data layer, `pkg/` for shared Go libraries,
`apps/<name>` for frontends) so there's something real to build the actual
domain on top of. There is no auth, no billing, no multi-tenancy, and no
replay-validation domain logic yet — just a health-checked API and a web app
that calls it.

## Layout

```
pkg/            shared Go libraries (http server, logging, errors, docs)
db/             ent client wiring + schemas (db/ent/schema is empty — no
                domain model yet, just a placeholder entity so the ent
                pipeline has something to generate)
services/api/   the Go API service (main.go, cmd/, app/, openapi/, pkg/)
scripts/        the ogen + openapi-typescript code generator
apps/web/       the React/Tamagui/Vite frontend
```

## Running it locally

### Backend

```sh
make dev        # go run ./services/api against a throwaway SQLite db in .data/
```

or, against Postgres in Docker:

```sh
make up         # docker compose up --build (postgres + api)
```

Either way, `GET /health` and `GET /ready` are live on `:8080`.

### Frontend

```sh
cd apps/web && pnpm install   # first time only
make web                      # pnpm --filter web dev, on :3000
```

The dev server proxies `/api/*` to `localhost:8080`, so the frontend and
backend can run side by side with no CORS setup.

### Codegen

```sh
make gen-ent     # regenerate the ent client after editing db/ent/schema
make gen-api     # regenerate the ogen server/client after editing openapi.yaml
```
