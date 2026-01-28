.PHONY: build test lint clean coverage

BINARY_NAME=tfbreak
BINARY_DIR=bin
GO=go
GOFLAGS=-trimpath
LDFLAGS=-s -w

# Build the binary
build:
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/tfbreak

# Run tests
test:
	$(GO) test -race -v ./...

# Run tests with coverage
coverage:
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run the binary
run: build
	./$(BINARY_DIR)/$(BINARY_NAME)
