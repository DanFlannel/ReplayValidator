##@ Core
.PHONY: help
help:  ## Display this help message.
	@echo "Usage:"
	@echo "  make [target]"
	@awk 'BEGIN {FS = ":.*?## "} \
		/^[a-zA-Z0-9_-]+:.*?## / { \
			printf "\033[36m  %-45s\033[0m %s\n", $$1, $$2 \
		} \
		/^##@/ { \
			printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
		}' $(MAKEFILE_LIST)

.PHONY: gen-ent
gen-ent: ## Generates all our files for our db schemas (ent)
	cd ./db/ent && go generate ./

.PHONY: gen-api
gen-api: ## Generate OpenAPI schemas using ogen
	@python3 scripts/api_generator/cli.py

.PHONY: gen-all
gen-all: gen-ent gen-api ## Generate ent + OpenAPI

##@ Development

.PHONY: dev
dev: ## Run the API locally against a throwaway SQLite db (no Postgres/Docker needed)
	DATABASE_DB_CONN="sqlite:./.data/dev.db?_fk=1" \
	ALLOWED_ORIGINS="http://localhost:3000" \
	go run ./services/api

.PHONY: web
web: ## Run the web app's Vite dev server
	pnpm --filter web dev

##@ Database

.PHONY: up
up: ## Start Postgres + the API in Docker (docker compose up --build)
	docker compose up --build

.PHONY: down
down: ## Stop the Docker stack
	docker compose down

##@ Testing

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: lint
lint: ## Run golangci-lint if installed
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, see https://golangci-lint.run/usage/install/"
