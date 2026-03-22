package zoneadd

import (
	"context"
	"strings"
	"testing"
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
