
.PHONY: help build test test-race linter changelog vendor-update vendor-clean vendor-bootstrap vendor-adminlte vendor-alpinejs docker-up docker-down docker-logs load-test-data docker-build docker-run docker-push

# Docker image
IMAGE_NAME ?= gopowerdns-admin
IMAGE_TAG  ?= latest

# Versions
BOOTSTRAP_VERSION := 5.3.8
ADMINLTE_VERSION := 4.0.0-rc7
JQUERY_VERSION := 4.0.0
DATATABLES_VERSION := 2.3.7
OVERLAYSCROLLBARS_VERSION := 2.14.0
BOOTSTRAP_ICONS_VERSION := 1.13.1
SOURCE_SANS_3_VERSION := 5.2.9
ALPINEJS_VERSION := 3.14.9

# Directories
VENDOR_DIR := internal/web/static/vendor
TMP_DIR := /tmp/gopowerdns-vendor

help:
	@echo "Available targets:"
	@echo ""
	@echo "Application Docker image:"
	@echo "  docker-build      - Build the application image (IMAGE_NAME, IMAGE_TAG)"
	@echo "  docker-run        - Run the application image locally"
	@echo "  docker-push       - Push the application image to a registry"
	@echo ""
	@echo "Dev Docker Compose:"
	@echo "  docker-up         - Start Docker Compose services"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-logs       - View Docker Compose logs"
	@echo "  load-test-data    - Load PowerDNS test data into running instance"
	@echo ""
	@echo "Development:"
	@echo "  build             - Build binary with version and branch baked in"
	@echo "  test              - Run all tests"
	@echo "  linter            - Run golangci-lint"
	@echo "  pre-commit        - Run pre-commit checks"
	@echo "  changelog         - Regenerate CHANGELOG.md via git-cliff"
	@echo ""
	@echo "Vendor Dependencies:"
	@echo "  vendor-update            - Update all vendor dependencies"
	@echo "  vendor-bootstrap         - Update only Bootstrap"
	@echo "  vendor-adminlte          - Update only AdminLTE"
	@echo "  vendor-jquery            - Update only jQuery"
	@echo "  vendor-datatables        - Update only DataTables"
	@echo "  vendor-overlayscrollbars - Update only OverlayScrollbars"
	@echo "  vendor-bootstrap-icons   - Update only Bootstrap Icons"
	@echo "  vendor-source-sans-3     - Update only Source Sans 3 font"
	@echo "  vendor-alpinejs          - Update only Alpine.js"
	@echo "  vendor-clean             - Remove all vendor dependencies"

build:
	@echo "Building..."
	@go build \
		-ldflags="-s -w \
			-X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.version=$(shell git describe --tags --always --dirty) \
			-X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.branch=$(shell git rev-parse --abbrev-ref HEAD)" \
		-o gopowerdns-admin .
	@echo "✓ Built: gopowerdns-admin"

test:
	@echo "Running tests..."
	@go test ./...
	@echo "✓ Tests passed"

linter:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "✓ Linter passed"

pre-commit: linter
	@echo "Running Pre-commit checks..."
	@pre-commit run --all-files
	@echo "✓ Pre-commit checks passed"

changelog:
	@echo "Generating CHANGELOG.md..."
	@git-cliff -o CHANGELOG.md
	@echo "✓ CHANGELOG.md updated"

vendor-update: vendor-bootstrap vendor-adminlte vendor-jquery vendor-datatables vendor-overlayscrollbars vendor-bootstrap-icons vendor-source-sans-3 vendor-alpinejs
	@echo "✓ All vendor dependencies updated"

vendor-bootstrap:
	@echo "Downloading Bootstrap $(BOOTSTRAP_VERSION)..."
	@mkdir -p $(TMP_DIR)
	@curl -sL https://github.com/twbs/bootstrap/releases/download/v$(BOOTSTRAP_VERSION)/bootstrap-$(BOOTSTRAP_VERSION)-dist.zip -o $(TMP_DIR)/bootstrap.zip
	@rm -rf $(VENDOR_DIR)/bootstrap-$(BOOTSTRAP_VERSION)-dist
	@unzip -q $(TMP_DIR)/bootstrap.zip -d $(VENDOR_DIR)/
	@rm $(TMP_DIR)/bootstrap.zip
	@echo "✓ Bootstrap $(BOOTSTRAP_VERSION) installed"

vendor-adminlte:
	@echo "Downloading AdminLTE $(ADMINLTE_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/adminlte-v4/css
	@mkdir -p $(VENDOR_DIR)/adminlte-v4/js
	@curl -sL https://cdn.jsdelivr.net/npm/admin-lte@$(ADMINLTE_VERSION)/dist/css/adminlte.min.css -o $(VENDOR_DIR)/adminlte-v4/css/adminlte.min.css
	@curl -sL https://cdn.jsdelivr.net/npm/admin-lte@$(ADMINLTE_VERSION)/dist/css/adminlte.css -o $(VENDOR_DIR)/adminlte-v4/css/adminlte.css
	@curl -sL https://cdn.jsdelivr.net/npm/admin-lte@$(ADMINLTE_VERSION)/dist/js/adminlte.min.js -o $(VENDOR_DIR)/adminlte-v4/js/adminlte.min.js
	@curl -sL https://cdn.jsdelivr.net/npm/admin-lte@$(ADMINLTE_VERSION)/dist/js/adminlte.js -o $(VENDOR_DIR)/adminlte-v4/js/adminlte.js
	@echo "✓ AdminLTE $(ADMINLTE_VERSION) installed"

vendor-jquery:
	@echo "Downloading jQuery $(JQUERY_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/jquery-$(JQUERY_VERSION)
	@curl -sL "https://cdn.jsdelivr.net/npm/jquery@$(JQUERY_VERSION)/dist/jquery.min.js" \
		-o $(VENDOR_DIR)/jquery-$(JQUERY_VERSION)/jquery.min.js
	@echo "✓ jQuery $(JQUERY_VERSION) installed"

vendor-datatables:
	@echo "Downloading DataTables $(DATATABLES_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/datatables-$(DATATABLES_VERSION)/css
	@mkdir -p $(VENDOR_DIR)/datatables-$(DATATABLES_VERSION)/js
	@curl -sL "https://cdn.datatables.net/$(DATATABLES_VERSION)/css/dataTables.bootstrap5.min.css" \
		-o $(VENDOR_DIR)/datatables-$(DATATABLES_VERSION)/css/dataTables.bootstrap5.min.css
	@curl -sL "https://cdn.datatables.net/$(DATATABLES_VERSION)/js/dataTables.min.js" \
		-o $(VENDOR_DIR)/datatables-$(DATATABLES_VERSION)/js/dataTables.min.js
	@curl -sL "https://cdn.datatables.net/$(DATATABLES_VERSION)/js/dataTables.bootstrap5.min.js" \
		-o $(VENDOR_DIR)/datatables-$(DATATABLES_VERSION)/js/dataTables.bootstrap5.min.js
	@echo "✓ DataTables $(DATATABLES_VERSION) installed"

vendor-overlayscrollbars:
	@echo "Downloading OverlayScrollbars $(OVERLAYSCROLLBARS_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/overlayscrollbars-$(OVERLAYSCROLLBARS_VERSION)/styles
	@mkdir -p $(VENDOR_DIR)/overlayscrollbars-$(OVERLAYSCROLLBARS_VERSION)/browser
	@curl -sL "https://cdn.jsdelivr.net/npm/overlayscrollbars@$(OVERLAYSCROLLBARS_VERSION)/styles/overlayscrollbars.min.css" \
		-o $(VENDOR_DIR)/overlayscrollbars-$(OVERLAYSCROLLBARS_VERSION)/styles/overlayscrollbars.min.css
	@curl -sL "https://cdn.jsdelivr.net/npm/overlayscrollbars@$(OVERLAYSCROLLBARS_VERSION)/browser/overlayscrollbars.browser.es6.min.js" \
		-o $(VENDOR_DIR)/overlayscrollbars-$(OVERLAYSCROLLBARS_VERSION)/browser/overlayscrollbars.browser.es6.min.js
	@echo "✓ OverlayScrollbars $(OVERLAYSCROLLBARS_VERSION) installed"

vendor-bootstrap-icons:
	@echo "Downloading Bootstrap Icons $(BOOTSTRAP_ICONS_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/bootstrap-icons-$(BOOTSTRAP_ICONS_VERSION)/font/fonts
	@curl -sL "https://cdn.jsdelivr.net/npm/bootstrap-icons@$(BOOTSTRAP_ICONS_VERSION)/font/bootstrap-icons.min.css" \
		-o $(VENDOR_DIR)/bootstrap-icons-$(BOOTSTRAP_ICONS_VERSION)/font/bootstrap-icons.min.css
	@curl -sL "https://cdn.jsdelivr.net/npm/bootstrap-icons@$(BOOTSTRAP_ICONS_VERSION)/font/fonts/bootstrap-icons.woff2" \
		-o $(VENDOR_DIR)/bootstrap-icons-$(BOOTSTRAP_ICONS_VERSION)/font/fonts/bootstrap-icons.woff2
	@curl -sL "https://cdn.jsdelivr.net/npm/bootstrap-icons@$(BOOTSTRAP_ICONS_VERSION)/font/fonts/bootstrap-icons.woff" \
		-o $(VENDOR_DIR)/bootstrap-icons-$(BOOTSTRAP_ICONS_VERSION)/font/fonts/bootstrap-icons.woff
	@echo "✓ Bootstrap Icons $(BOOTSTRAP_ICONS_VERSION) installed"

vendor-source-sans-3:
	@echo "Downloading Source Sans 3 $(SOURCE_SANS_3_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/source-sans-3-$(SOURCE_SANS_3_VERSION)/files
	@curl -sL "https://cdn.jsdelivr.net/npm/@fontsource/source-sans-3@$(SOURCE_SANS_3_VERSION)/index.css" \
		-o $(VENDOR_DIR)/source-sans-3-$(SOURCE_SANS_3_VERSION)/index.css
	@grep -oE '\./files/[^)"]+' $(VENDOR_DIR)/source-sans-3-$(SOURCE_SANS_3_VERSION)/index.css | \
		sort -u | sed 's|\./||' | while read -r file; do \
		curl -sL "https://cdn.jsdelivr.net/npm/@fontsource/source-sans-3@$(SOURCE_SANS_3_VERSION)/$$file" \
			-o "$(VENDOR_DIR)/source-sans-3-$(SOURCE_SANS_3_VERSION)/$$file"; \
	done
	@echo "✓ Source Sans 3 $(SOURCE_SANS_3_VERSION) installed"

vendor-alpinejs:
	@echo "Downloading Alpine.js $(ALPINEJS_VERSION)..."
	@mkdir -p $(VENDOR_DIR)/alpinejs-$(ALPINEJS_VERSION)
	@curl -sL "https://cdn.jsdelivr.net/npm/alpinejs@$(ALPINEJS_VERSION)/dist/cdn.min.js" \
		-o $(VENDOR_DIR)/alpinejs-$(ALPINEJS_VERSION)/alpine.min.js
	@echo "✓ Alpine.js $(ALPINEJS_VERSION) installed"

vendor-clean:
	@echo "Cleaning vendor dependencies..."
	@rm -rf $(VENDOR_DIR)/bootstrap-*
	@rm -rf $(VENDOR_DIR)/adminlte-*
	@rm -rf $(VENDOR_DIR)/jquery-*
	@rm -rf $(VENDOR_DIR)/datatables-*
	@rm -rf $(VENDOR_DIR)/overlayscrollbars-*
	@rm -rf $(VENDOR_DIR)/bootstrap-icons-*
	@rm -rf $(VENDOR_DIR)/source-sans-3-*
	@rm -rf $(VENDOR_DIR)/alpinejs-*
	@echo "✓ Vendor dependencies cleaned"

docker-build:
	@echo "Building $(IMAGE_NAME):$(IMAGE_TAG)..."
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "✓ Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

docker-run:
	@echo "Running $(IMAGE_NAME):$(IMAGE_TAG)..."
	@docker run --rm \
		-p 8080:8080 \
		-v "$(PWD)/etc:/etc/go-pdns:ro" \
		-v gopowerdns-data:/var/lib/go-pdns \
		$(IMAGE_NAME):$(IMAGE_TAG)

docker-push:
	@echo "Pushing $(IMAGE_NAME):$(IMAGE_TAG)..."
	@docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "✓ Pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

docker-up:
	@echo "Starting Docker Compose services..."
	@docker-compose up -d
	@echo "✓ Services started"
	@echo ""
	@echo "Services:"
	@echo "  PostgreSQL:   localhost:5432"
	@echo "  MySQL:        localhost:3306"
	@echo "  PowerDNS API: http://localhost:8081"
	@echo "  PowerDNS DNS: localhost:53"

docker-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose down
	@echo "✓ Services stopped"

docker-logs:
	@docker-compose logs -f

load-test-data:
	@echo "Loading PowerDNS test data..."
	@./docker/pdns/load-test-data.py
	@echo ""
	@echo "✓ Test data loaded successfully"
