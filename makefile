include .env
export

export PROJECT_ROOT=${shell pwd}
export LOGGER_FOLDER=${PROJECT_ROOT}/log

.PHONY: tests
.PHONY: tests-coverage
.PHONY: unit-tests
local-up:
	docker compose up -d --build postgres && \
	docker compose up -d --build kafka && \
	go run ${PROJECT_ROOT}/cmd/app/main.go

env-up:
	docker compose up -d --build

env-down:
	docker compose down

migrate-up:
	@migrate \
	-path ${PROJECT_ROOT}/migrations \
	-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable	\
	up

migrate-down:
	@migrate \
	-path ${PROJECT_ROOT}/migrations \
	-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable	\
	down

tests:
	go test -count=1 -v ./...


unit-tests:
	go test -count=1 -v ./internal/...

tests-coverage:
	go test -count=1 -v -covermode=atomic -coverprofile=coverage.out ./...

open-coverage:
	go tool cover -html=coverage.out -o coverage.html