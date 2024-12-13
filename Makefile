export GO111MODULE=on

project_root:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

APP=wordle
BIN_DIR="$(project_root)/bin"
BUILD_DIR="$(project_root)/build"
APP_EXECUTABLE="$(BIN_DIR)/$(APP)"

SHELL := /bin/bash

lint:
	golangci-lint run --enable-all

vet:
	go vet ./...

fmt:
	go fmt ./...

check-quality: lint fmt vet

tidy:
	go mod tidy

vendor:
	go mod vendor

mod: tidy vendor

test:
	mkdir -p $(BUILD_DIR)
	go test -v -race -timeout 1m ./... -coverprofile=$(BUILD_DIR)/coverage.out -json

test-report:
	mkdir -p $(BUILD_DIR)
	go test -v -race -timeout 1m ./... -coverprofile=$(BUILD_DIR)/coverage.out -json > $(BUILD_DIR)/report.json

coverage: test
	mkdir -p $(BUILD_DIR)
	go tool cover -html=$(BUILD_DIR)/coverage.out

build:
	mkdir -p $(BIN_DIR)
	cd $(project_root)/pkg/main; go build -mod vendor -o $(APP_EXECUTABLE)

exec: build
	$(APP_EXECUTABLE)

clean:
	go clean
	rm -rf $(BIN_DIR)
	rm -f $(BUILD_DIR)/coverage.out

.PHONY: build vendor

 all: check-quality test build

.PHONY: help