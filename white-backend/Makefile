SHELL := /bin/bash
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(PATH):$(GOBIN)

SERVICES := identity integrations publishing seosuite scheduler gateway

.PHONY: tools proto sqlc generate build test lint vet migrate-% run-% up down fmt

## Install code generators into GOPATH/bin (protoc itself comes from Homebrew/apt).
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

proto:
	./scripts/proto.sh

## sqlc reads every internal/<svc>/adapter/postgres/sqlc.yaml.
sqlc:
	@for f in $$(find internal -name sqlc.yaml); do (cd $$(dirname $$f) && sqlc generate); done

generate: proto sqlc

build:
	@mkdir -p bin
	@for s in $(SERVICES); do go build -o bin/$$s ./cmd/$$s || exit 1; done

## Integration tests run when WHITE_TEST_DATABASE_URL is set. Packages of one
## service share a schema and truncate it, so test packages run one at a time.
test:
	go test -p 1 ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -l -w cmd internal pkg

## make migrate-identity (runs goose up for one service against DATABASE_URL)
migrate-%:
	go run ./cmd/$* migrate

## make run-identity
run-%:
	go run ./cmd/$* serve

up:
	docker compose -f deploy/docker-compose.yml up --build -d

down:
	docker compose -f deploy/docker-compose.yml down
