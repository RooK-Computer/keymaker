BIN_DIR := bin
BINARY := $(BIN_DIR)/keymaker

.PHONY: all build clean run test-build deps

all: build

build: deps
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY)"
	@go build -o $(BINARY) ./

run: build
	@$(BINARY)

clean:
	@rm -rf $(BIN_DIR)

# Quick compile and run check
test-build: build
	@echo "Running $(BINARY)"
	@$(BINARY)

# Fetch and tidy Go module dependencies
deps:
	@echo "Fetching Go module dependencies"
	@go mod tidy
