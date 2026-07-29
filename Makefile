BIN   := bin/api
GO    := go
LINT  := golangci-lint

.PHONY: all test test-race test-unit vet lint build clean

all: vet build

## test  — run all tests including integration (requires Docker).
test:
	$(GO) test -count=1 ./...

## test-race  — run all tests with the race detector (catches data races).
test-race:
	$(GO) test -race -count=1 ./...

## test-unit  — run unit tests only (domain, usecase, handler, no Docker needed).
test-unit:
	$(GO) test -count=1 \
		./internal/domain/... \
		./internal/usecase/... \
		./internal/handler/... \
		./cmd/api/...

## vet  — run go vet on all packages.
vet:
	$(GO) vet ./...

## lint  — run golangci-lint (install: brew install golangci-lint or go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest).
lint:
	$(LINT) run

## swagger  — regenerate OpenAPI docs (install: go install github.com/swaggo/swag/cmd/swag@latest).
swagger:
	swag init -g cmd/api/main.go --output docs/

## build  — compile the API binary.
build:
	$(GO) build -o $(BIN) ./cmd/api

## clean  — remove compiled artifacts.
clean:
	rm -rf bin/
