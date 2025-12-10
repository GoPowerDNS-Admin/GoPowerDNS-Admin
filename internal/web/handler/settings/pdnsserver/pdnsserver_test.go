package pdnsserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	controller "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/pdnsserver"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to create test database")

	// Migrate the schema
	err = db.AutoMigrate(&models.Setting{})
	require.NoError(t, err, "failed to migrate test database")

	return db
}

func TestService_Get_WithExistingSettings(t *testing.T) {
	db := setupTestDB(t)

	service := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	// Save test settings
	settings := &controller.Settings{
		APIServerURL: "https://pdns.example.com:8081",
		APIKey:       "test-api-key",
		VHost:        "4.8.0",
	}
	err := settings.Save(db)
	require.NoError(t, err)

	// Create Fiber app with custom template engine
	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Get("/settings/pdns-server", service.Get)

	// Create request and test
	req := httptest.NewRequest(http.MethodGet, "/settings/pdns-server", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// The mock template engine returns 200 OK when Settings is present
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestService_Get_WithoutSettings(t *testing.T) {
	db := setupTestDB(t)

	service := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Get("/settings/pdns-server", service.Get)

	// Create request and test
	req := httptest.NewRequest(http.MethodGet, "/settings/pdns-server", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Should handle missing settings gracefully (render empty form)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestService_Get_WithNilDatabase(t *testing.T) {
	service := &Service{
		cfg:       &config.Config{},
		db:        nil,
		validator: validator.New(),
	}

	app := fiber.New()
	app.Get("/settings/pdns-server", service.Get)

	req := httptest.NewRequest(http.MethodGet, "/settings/pdns-server", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Should return internal server error
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestService_Post_Success(t *testing.T) {
	db := setupTestDB(t)

	service := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Post("/settings/pdns-server", service.Post)

	// Create form data
	formData := "api_server_url=https://pdns.example.com:8081&api_key=test-key-123&version=4.9.0"
	req := httptest.NewRequest(http.MethodPost, "/settings/pdns-server", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verify settings were saved to database
	loaded := &controller.Settings{}
	err = loaded.Load(db)
	require.NoError(t, err)
	assert.Equal(t, "https://pdns.example.com:8081", loaded.APIServerURL)
	assert.Equal(t, "test-key-123", loaded.APIKey)
	assert.Equal(t, "4.9.0", loaded.VHost)
}

func TestService_Post_InvalidFormData(t *testing.T) {
	db := setupTestDB(t)

	service := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Post("/settings/pdns-server", service.Post)

	// Send empty form data - should fail validation (required fields)
	formData := ""
	req := httptest.NewRequest(http.MethodPost, "/settings/pdns-server", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Should fail validation and return bad request
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestService_Post_InvalidAPIKey(t *testing.T) {
	db := setupTestDB(t)

	service := &Service{
		cfg:       &config.Config{},
		db:        db,
		validator: validator.New(),
	}

	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Post("/settings/pdns-server", service.Post)

	// Send form data with API key that is too short (less than 8 chars)
	formData := "api_server_url=https://pdns.example.com&api_key=short&version=4.9.0"
	req := httptest.NewRequest(http.MethodPost, "/settings/pdns-server", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Should fail validation and return bad request
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestService_Post_DatabaseError(t *testing.T) {
	// Using nil database to trigger save error
	service := &Service{
		cfg:       &config.Config{},
		db:        nil,
		validator: validator.New(),
	}

	app := fiber.New(fiber.Config{
		Views: &mockTemplateEngine{},
	})

	app.Post("/settings/pdns-server", service.Post)

	// Create valid form data with API key that passes validation (at least 8 characters)
	formData := "api_server_url=https://pdns.example.com&api_key=validkey123&version=4.9.0"
	req := httptest.NewRequest(http.MethodPost, "/settings/pdns-server", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Should return internal server error
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// mockTemplateEngine is a simple mock for testing.
type mockTemplateEngine struct{}

func (m *mockTemplateEngine) Load() error {
	return nil
}

func (m *mockTemplateEngine) Render(_ io.Writer, _ string, binding interface{}, _ ...string) error {
	// Check that Settings is in the binding
	if data, ok := binding.(fiber.Map); ok {
		if _, hasSettings := data["Settings"]; hasSettings {
			return nil
		}
	}
	return fiber.ErrInternalServerError
}
