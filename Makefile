# Copyright (c) 2026 H0llyW00dzZ All rights reserved.
#
# By accessing or using this software, you agree to be bound by the terms
# of the License Agreement, which you can find at LICENSE files.

.PHONY: test test-short test-cover test-verbose build clean

# Run tests with race detector (mirrors CI).
test:
	go test -race ./src/...

# Run tests with verbose output and race detector.
test-verbose:
	go test -v -race ./src/...

# Run tests with coverage report and race detector (same as CI).
test-cover:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./src/...
	@echo ""
	@echo "Coverage report written to coverage.txt"
	@echo "View in browser: go tool cover -html=coverage.txt"

# Run unit tests only (skip live DNS tests).
test-short:
	go test -race -short ./src/...

# Build all packages.
# this disabled for now, because its a library
# build:
# 	go build ./src/...

# Remove generated files.
clean:
	rm -f coverage.txt
