package zoneedit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// noopViews is a minimal Fiber view engine stub that always succeeds.
type noopViews struct{}

func (n *noopViews) Load() error { return nil }
func (n *noopViews) Render(_ io.Writer, _ string, _ interface{}, _ ...string) error {
	return nil
}

// Test that Get short-circuits safely when PDNS client is not initialized.
func TestGet_ReturnsEarlyWhenPDNSClientNil(t *testing.T) {
	// Ensure PDNS client is nil to trigger the error-render path
	powerdns.Engine.Client = nil

	app := fiber.New(fiber.Config{Views: &noopViews{}})

	svc := &Service{}
	app.Get("/zones/:name/edit", svc.Get)

	req := httptest.NewRequest(fiber.MethodGet, "/zones/example.com/edit", http.NoBody)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// When PDNS client is nil, handler sets 500 and renders an error.
	// The important assertion is that there is no panic (nil deref).
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected status 500 when PDNS client is nil, got %d", resp.StatusCode)
	}
}
