BIN_DIR := bin
BINARY := $(BIN_DIR)/keymaker
SIM_BINARY := $(BIN_DIR)/keymaker-sim

SIM_PORT ?= 8080
SIM_LISTEN ?= :$(SIM_PORT)
VITE_API_BASE_URL ?= http://127.0.0.1:$(SIM_PORT)

WEBAPP_DIR := webapp
WEBAPP_OUT := internal/assets/web
WEBAPP_INDEX := $(WEBAPP_OUT)/index.html
WEBAPP_INSTALL_STAMP := $(WEBAPP_DIR)/node_modules/.installed
WEBAPP_SOURCES := $(shell find $(WEBAPP_DIR)/src -type f -print) \
	$(WEBAPP_DIR)/index.html \
	$(WEBAPP_DIR)/vite.config.ts \
	$(WEBAPP_DIR)/tsconfig.json \
	$(WEBAPP_DIR)/tsconfig.node.json \
	$(WEBAPP_DIR)/package.json \
	$(WEBAPP_DIR)/package-lock.json \
	api-spec/openapi.yaml

.PHONY: all build build-sim clean run sim sim-dev test-build deps api-docs webapp-build web-install web-dev dev validate-sim-api

all: build

build: deps webapp-build
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY)"
	@go build -o $(BINARY) ./

build-sim: deps webapp-build
	@mkdir -p $(BIN_DIR)
	@echo "Building $(SIM_BINARY)"
	@go build -o $(SIM_BINARY) ./simulator

run: build
	@$(BINARY)

sim: build-sim
	@$(SIM_BINARY)

sim-dev: build-sim
	@$(SIM_BINARY) --listen $(SIM_LISTEN) --dev

clean:
	@rm -rf $(BIN_DIR)

webapp-build: $(WEBAPP_INDEX)

web-install: $(WEBAPP_INSTALL_STAMP)
	@true

web-dev: web-install
	@cd $(WEBAPP_DIR) && VITE_API_BASE_URL=$(VITE_API_BASE_URL) npm run dev

# Run simulator + web UI dev server together.
dev:
	@$(MAKE) --output-sync=target -j2 sim-dev web-dev

validate-sim-api:
	@bash ./tools/validate_sim_api.sh

$(WEBAPP_INSTALL_STAMP): $(WEBAPP_DIR)/package-lock.json $(WEBAPP_DIR)/package.json
	@echo "Installing webapp dependencies"
	@cd $(WEBAPP_DIR) && npm ci
	@mkdir -p $(WEBAPP_DIR)/node_modules
	@: > $(WEBAPP_INSTALL_STAMP)

$(WEBAPP_INDEX): $(WEBAPP_INSTALL_STAMP) $(WEBAPP_SOURCES)
	@echo "Building web app into $(WEBAPP_OUT)"
	@cd $(WEBAPP_DIR) && npm run build

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
