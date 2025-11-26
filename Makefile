# CoreNVR Makefile

BINARY_NAME=corenvr
VERSION=0.1.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build build-linux build-pi clean deps test help

all: build

## build: Build for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/corenvr

## build-linux: Build for Linux (amd64)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/corenvr

## build-pi: Build for Raspberry Pi (arm64)
build-pi:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/corenvr

## build-pi32: Build for Raspberry Pi (32-bit armv7)
build-pi32:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-armv7 ./cmd/corenvr

## build-all: Build for all platforms
build-all: build build-linux build-pi build-pi32

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-*

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## test: Run tests
test:
	$(GOTEST) -v ./...

## hashpass: Generate a password hash
hashpass:
	@read -p "Enter password: " pwd; \
	$(GOCMD) run tools/hashpass.go "$$pwd"

## help: Show this help message
help:
	@echo "CoreNVR - Lightweight NVR for Raspberry Pi"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
