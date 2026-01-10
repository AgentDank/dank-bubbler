.PHONY: help build clean test install lint fmt

help:
	@echo "dank-bubbler build targets:"
	@echo "  build          - Build all tools"
	@echo "  build-demo     - Build brand-demo tool"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install tools into $$GOPATH/bin"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy go.mod"

build: build-demo

build-demo:
	@echo "Building brand-demo..."
	@go build -o ./bin/db-brand-demo ./cmd/brand-demo

clean:
	@rm -rf ./bin
	@go clean

install: build
	@go install ./cmd/brand-demo

test:
	@go test -v ./...

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...
	@gofmt -s -w .

tidy:
	@go mod tidy
