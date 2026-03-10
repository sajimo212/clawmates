PROTO_FILES=proto/core.proto proto/matching.proto

.PHONY: proto
proto:
	protoc --go_out=. --go_opt=module=clawmates/backend \
		--go-grpc_out=. --go-grpc_opt=module=clawmates/backend \
		$(PROTO_FILES)

.PHONY: test
test:
	GOCACHE=$(PWD)/.gocache GOMODCACHE=$(PWD)/.gomodcache go test ./...

.PHONY: run-gateway
run-gateway:
	go run ./cmd/gateway

.PHONY: run-core
run-core:
	go run ./cmd/core-service

.PHONY: run-matching
run-matching:
	go run ./cmd/matching-service
