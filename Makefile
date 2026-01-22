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
	@echo "Generating offline Swagger UI docs under docs/api"
	@cd api-spec && npm ci
	@rm -rf docs/api
	@mkdir -p docs/api
	@cp -r api-spec/node_modules/swagger-ui-dist/* docs/api/
	@cp api-spec/openapi.yaml docs/api/openapi.yaml
	@cp api-spec/swagger-ui-index.html docs/api/index.html
	@mkdir -p docs
	@test -f docs/.nojekyll || : > docs/.nojekyll
	@echo "Done: docs/api/index.html"
