package zoneadd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pdnsapi "github.com/joeig/go-powerdns/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

func TestResolveZoneName_Forward_AddsTrailingDot(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeForward, Name: "example.com"}

	if err := resolveZoneName(form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.Name != "example.com." {
		t.Errorf("expected trailing dot, got %q", form.Name)
	}
}

func TestResolveZoneName_Forward_AlreadyHasDot(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeForward, Name: "example.com."}

	if err := resolveZoneName(form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.Name != "example.com." {
		t.Errorf("expected name unchanged, got %q", form.Name)
	}
}

func TestResolveZoneName_ReverseIPv4_Valid(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeReverseIPv4, ReverseNetwork: "192.168.1.0/24"}

	if err := resolveZoneName(form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(form.Name, ".in-addr.arpa.") {
		t.Errorf("expected .in-addr.arpa. suffix, got %q", form.Name)
	}
}

func TestResolveZoneName_ReverseIPv4_Invalid(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeReverseIPv4, ReverseNetwork: "not-a-cidr"}

	if err := resolveZoneName(form); err == nil {
		t.Fatal("expected error for invalid IPv4 CIDR, got nil")
	}
}

func TestResolveZoneName_ReverseIPv6_Valid(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeReverseIPv6, ReverseNetwork: "2001:db8::/32"}

	if err := resolveZoneName(form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(form.Name, ".ip6.arpa.") {
		t.Errorf("expected .ip6.arpa. suffix, got %q", form.Name)
	}
}

func TestResolveZoneName_ReverseIPv6_Invalid(t *testing.T) {
	form := &ZoneForm{ZoneType: ZoneTypeReverseIPv6, ReverseNetwork: "not-a-cidr"}

	if err := resolveZoneName(form); err == nil {
		t.Fatal("expected error for invalid IPv6 CIDR, got nil")
	}
}

func TestCreateZone_UnknownKind(t *testing.T) {
	form := &ZoneForm{Kind: ZoneKind("Unknown"), Name: "test.", SOAEditAPI: SOAEditAPIDefault}

	err := createZone(context.Background(), form)
	if err == nil {
		t.Fatal("expected error for unknown zone kind, got nil")
	}

	if !strings.Contains(err.Error(), "Unknown") {
		t.Errorf("expected error to mention kind name, got %q", err.Error())
	}
}

func TestCreateZone_EnablesAPIRectify(t *testing.T) {
	for _, kind := range []ZoneKind{ZoneKindNative, ZoneKindMaster} {
		t.Run(string(kind), func(t *testing.T) {
			got := make(chan map[string]any, 1)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any

				_ = json.NewDecoder(r.Body).Decode(&body)
				got <- body

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"name":"example.com."}`))
			}))
			defer srv.Close()

			powerdns.Engine.Client = pdnsapi.New(srv.URL, "localhost")

			t.Cleanup(func() { powerdns.Engine.Client = nil })

			form := &ZoneForm{Kind: kind, Name: "example.com.", SOAEditAPI: SOAEditAPIDefault}
			if err := createZone(context.Background(), form); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if body := <-got; body["api_rectify"] != true {
				t.Errorf("expected api_rectify true, got %v", body["api_rectify"])
			}
		})
	}
}
