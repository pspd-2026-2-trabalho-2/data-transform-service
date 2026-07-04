# data-transform-service — atalhos de build/dev.
# No Windows sem `make`, rode os comandos equivalentes à mão (ver README).

MODULE := github.com/pspd-2026-2-trabalho-2/data-transform-service
BIN    := bin/data-transform-service
# Caminho dos well-known types do protoc (google/protobuf/struct.proto).
# Windows (zip oficial): C:/protoc/include · Linux: normalmente /usr/include
PROTOC_INCLUDE ?= C:/protoc/include

.PHONY: proto build run test docker compose-up compose-down tidy tools

## Gera o código Go: mensagens (datatransform + patientdata) e o service gRPC do datatransform.
proto:
	protoc -I proto -I "$(PROTOC_INCLUDE)" \
	  --go_out=. --go_opt=module=$(MODULE) \
	  datatransform/v1/datatransform.proto patientdata/v1/patientdata.proto
	protoc -I proto -I "$(PROTOC_INCLUDE)" \
	  --go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
	  datatransform/v1/datatransform.proto

## Instala os plugins protoc do Go
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

build:
	go build -o $(BIN) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

docker:
	docker build -t data-transform-service:local .

## Sobe o e2e local: Postgres + patient-data + data-transform
compose-up:
	docker compose up -d --build

compose-down:
	docker compose down -v

tidy:
	go mod tidy
