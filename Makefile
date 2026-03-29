.PHONY: build install clean test help init

# Binary name
BINARY_NAME=caddy-dns-sync

# Build variables
VERSION?=1.0.0
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

# Go build flags
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/caddy-dns-sync
	@echo "Built: $(BINARY_NAME)"

build-all: ## Build for all supported platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/caddy-dns-sync
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/caddy-dns-sync
	@echo "Binaries built in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

install: build ## Build and install the binary (requires sudo)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@sudo install -m 755 $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(BINARY_NAME)"

init: install ## Install and initialize the system (requires sudo)
	@echo "Initializing system..."
	@sudo $(INSTALL_DIR)/$(BINARY_NAME) --init
	@echo "System initialized. Edit /etc/caddy-dns-sync/provider.conf to configure providers."

uninstall: ## Remove the installed binary
	@echo "Removing $(INSTALL_DIR)/$(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled."

clean: ## Remove built binaries
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Run formatters and linters

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

run: ## Run the application (requires config)
	@echo "Running $(BINARY_NAME)..."
	@go run ./cmd/caddy-dns-sync $(ARGS)

release: clean build-all ## Prepare a release build
	@echo "Release build complete!"
	@echo "Binaries in $(BUILD_DIR)/"
	@cd $(BUILD_DIR) && sha256sum * > checksums.txt
	@cat $(BUILD_DIR)/checksums.txt
