# SoulNexus build helpers

.PHONY: proto proto-auth build test test-cover lint run-server run-voice web-dev web-build admin-dev admin-build docker-build docker-up docker-down migrate seed install clean

PROTOC ?= protoc
AUTH_PROTO = internal/grpc/auth/proto/auth/v1/auth.proto
AUTH_PROTO_OUT = internal/grpc/auth/pb
AUTH_PROTO_INCLUDE = internal/grpc/auth/proto

# Regenerate auth gRPC / protobuf Go code after editing auth.proto.
proto: proto-auth

proto-auth:
	$(PROTOC) \
		--go_out=$(AUTH_PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(AUTH_PROTO_OUT) --go-grpc_opt=paths=source_relative \
		-I $(AUTH_PROTO_INCLUDE) \
		$(AUTH_PROTO)

build:
	go build -o bin/server ./cmd/server
	go build -o bin/voice ./cmd/voice

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

run-server:
	go run ./cmd/server

run-voice:
	go run ./cmd/voice

# Frontend targets
web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

admin-dev:
	cd admin && npm run dev

admin-build:
	cd admin && npm run build

# Docker targets (requires Dockerfile and docker-compose.yml)
docker-build:
	docker build -t soulnexus-server -f Dockerfile.server .
	docker build -t soulnexus-voice -f Dockerfile.voice .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Database migration (uses GORM AutoMigrate via -init flag)
migrate:
	go run ./cmd/server -init

# Seed database with demo data
seed:
	go run ./cmd/server -seed

# Install Go dependencies
install:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache
