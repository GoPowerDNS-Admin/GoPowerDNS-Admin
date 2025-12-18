.PHONY: help test linter vendor-update vendor-clean vendor-bootstrap vendor-adminlte docker-up docker-down docker-logs load-test-data

# Versions
BOOTSTRAP_VERSION := 5.3.8
ADMINLTE_VERSION := 4.0.0-rc4

# Directories
VENDOR_DIR := internal/web/static/vendor
TMP_DIR := /tmp/gopowerdns-vendor

help:
	@echo "Available targets:"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up         - Start Docker Compose services"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  docker-logs       - View Docker Compose logs"
	@echo "  load-test-data    - Load PowerDNS test data into running instance"
	@echo ""
	@echo "Development:"
	@echo "  test              - Run all tests"
	@echo "  linter            - Run golangci-lint"
	@echo "  pre-commit        - Run pre-commit checks"
	@echo ""
	@echo "Vendor Dependencies:"
	@echo "  vendor-update     - Update all vendor dependencies"
	@echo "  vendor-bootstrap  - Update only Bootstrap"
	@echo "  vendor-adminlte   - Update only AdminLTE"
	@echo "  vendor-clean      - Remove all vendor dependencies"

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

vendor-update: vendor-bootstrap vendor-adminlte
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

vendor-clean:
	@echo "Cleaning vendor dependencies..."
	@rm -rf $(VENDOR_DIR)/bootstrap-*
	@rm -rf $(VENDOR_DIR)/adminlte-*
	@echo "✓ Vendor dependencies cleaned"

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
