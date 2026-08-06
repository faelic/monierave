GO ?= go

compose-up:
	docker compose up --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs --follow api relay worker

migrateup:
	@test -n "$(DB_SOURCE)" || (echo "DB_SOURCE is required" && exit 1)
	migrate -path db/migration -database "$(DB_SOURCE)" -verbose up

migratedown1:
	@test -n "$(DB_SOURCE)" || (echo "DB_SOURCE is required" && exit 1)
	migrate -path db/migration -database "$(DB_SOURCE)" -verbose down 1

migration-check:
	./scripts/check-migrations.sh

sqlc:
	sqlc generate

generated-check:
	./scripts/check-generated.sh

mock:
	mockgen -destination db/mock/store.go -package mockdb github.com/faelic/monierave/db/sqlc Store

format:
	gofmt -w $$(git ls-files '*.go')

format-check:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))"

vet:
	$(GO) vet ./...

test:
	MAILER_PROVIDER=log $(GO) test -cover ./...

test-race:
	MAILER_PROVIDER=log $(GO) test -race -cover ./...

test-invariants:
	./scripts/test-financial-invariants.sh

server:
	$(GO) run main.go api

relay:
	$(GO) run main.go relay

worker:
	$(GO) run main.go worker

db-docs:
	dbdocs build doc/db.dbml

db-schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

ci: format-check vet generated-check migration-check test-invariants test-race

.PHONY: compose-up compose-down compose-logs migrateup migratedown1 \
	migration-check sqlc generated-check mock format format-check vet test \
	test-race test-invariants server relay worker db-docs db-schema ci
