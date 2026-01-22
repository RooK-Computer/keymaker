BIN_DIR := bin
BINARY := $(BIN_DIR)/keymaker

.PHONY: all build clean run test-build deps api-docs

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

# Regenerate offline API docs (OpenAPI YAML -> Swagger UI static folder)
api-docs:
	@echo "Generating offline Swagger UI docs under api-spec/docs"
	@cd api-spec && npm ci
	@rm -rf api-spec/docs
	@mkdir -p api-spec/docs
	@cp -r api-spec/node_modules/swagger-ui-dist/* api-spec/docs/
	@cp api-spec/openapi.yaml api-spec/docs/openapi.yaml
	@cp api-spec/swagger-ui-index.html api-spec/docs/index.html
	@echo "Done: api-spec/docs/index.html"
