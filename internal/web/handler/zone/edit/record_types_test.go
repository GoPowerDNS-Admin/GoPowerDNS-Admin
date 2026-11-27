package zoneedit

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	zonesettings "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/zone"
)

func newRecordTypeService(t *testing.T, records config.Record) *Service {
	t.Helper()

	db := newTestDB(t)

	settings := zonesettings.RecordSettings{Records: records}
	if err := settings.Save(db); err != nil {
		t.Fatalf("save record settings: %v", err)
	}

	return &Service{db: db}
}

func TestValidateRecordsUpdateAreValidTypes(t *testing.T) {
	svc := newRecordTypeService(t, config.Record{
		"A": {Forward: true},
	})

	tests := []struct {
		name    string
		change  RecordChange
		wantErr string
	}{
		{"allowed type", RecordChange{Changed: true, Type: "A"}, ""},
		{"disallowed new type", RecordChange{Changed: true, Type: "TXT"}, "Modification of record type TXT is not allowed"},
		{"disallowed type that already existed", RecordChange{Existed: true, Changed: true, Type: "SOA"}, ""},
		{"type allowed only for reverse zones", RecordChange{Changed: true, Type: "PTR"},
			"Modification of record type PTR is not allowed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := &RecordsUpdateRequest{Changes: []RecordChange{tc.change}}

			err := svc.validateRecordsUpdateAreValidTypes("example.com.", request, false)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}

				return
			}

			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// A disallowed record type must stop the request before it reaches PowerDNS.
// With the PowerDNS client unset, a handler that wrongly continues answers 500
// instead of 400.
func TestPostRecords_RejectsDisallowedTypeBeforePowerDNS(t *testing.T) {
	powerdns.Engine.Client = nil

	svc := newRecordTypeService(t, config.Record{
		"A": {Forward: true},
	})

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("CurrentUser", models.User{ID: 1})
		return c.Next()
	})
	app.Post("/zone/edit/:name/records", svc.PostRecords)

	body := `{"changes":[{"changed":true,"name":"example.com.","type":"TXT","ttl":3600,` +
		`"records":[{"content":"\"hello\""}]}]}`
	req := httptest.NewRequestWithContext(context.Background(), fiber.MethodPost,
		"/zone/edit/example.com/records", strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("want status 400, got %d: %s", resp.StatusCode, raw)
	}

	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode response %q: %v", raw, err)
	}

	if payload.Success || payload.Message != "Modification of record type TXT is not allowed" {
		t.Fatalf("unexpected response: %+v", payload)
	}
}
