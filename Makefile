# SoulNexus build helpers

.PHONY: proto proto-auth build test

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
	go build ./...

test:
	go test ./...
