APP_NAME ?= sync-service
MAIN_PKG := ./cmd/service
BUILD_DIR := bin
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all run test testv cover build clean tidy fmt vet lint mod graph

all: test build

run:
	@go run $(MAIN_PKG) $(ARGS)

test:
	@go test ./...

cover:
	@go test -cover -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1
	@echo "HTML: go tool cover -html=coverage.out"

build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "Build: $(BUILD_DIR)/$(APP_NAME)"

clean:
	@rm -rf $(BUILD_DIR) coverage.out

tidy:
	@go mod tidy

fmt:
	@go fmt ./...

vet:
	@go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "Missing golangci-lint"

imports:
	@command -v goimports >/dev/null 2>&1 && goimports -w $(GO_FILES) || echo "Missing goimports (go install golang.org/x/tools/cmd/goimports@latest)"

mod:
	@go list -m all

graph:
	@go mod graph
