package zoneedit

import (
	"testing"

	gormsqlite "github.com/glebarez/sqlite"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormsqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory DB: %v", err)
	}

	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	return db
}

func strPtr(s string) *string { return &s }
func rrType(s string) *pdnsapi.RRType {
	r := pdnsapi.RRType(s)
	return &r
}

// ── ipv4PTRName ───────────────────────────────────────────────────────────────

func TestIPv4PTRName(t *testing.T) {
	tests := []struct {
		ip   string
		want string
		err  bool
	}{
		{"192.0.2.1", "1.2.0.192.in-addr.arpa.", false},
		{"10.0.0.1", "1.0.0.10.in-addr.arpa.", false},
		{"255.255.255.255", "255.255.255.255.in-addr.arpa.", false},
		{"0.0.0.0", "0.0.0.0.in-addr.arpa.", false},
		{"not-an-ip", "", true},
		{"2001:db8::1", "", true}, // IPv6 must be rejected
	}

	for _, tc := range tests {
		t.Run(tc.ip, func(t *testing.T) {
			got, err := ipv4PTRName(tc.ip)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", tc.ip, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

// ── ipv6PTRName ───────────────────────────────────────────────────────────────

func TestIPv6PTRName(t *testing.T) {
	tests := []struct {
		ip   string
		want string
		err  bool
	}{
		{
			"2001:db8::1",
			"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.",
			false,
		},
		{
			"::1",
			"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa.",
			false,
		},
		{
			"2a02:d58:2:2000::1",
			"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.2.0.0.0.8.5.d.0.2.0.a.2.ip6.arpa.",
			false,
		},
		{"not-an-ip", "", true},
		{"192.0.2.1", "", true}, // IPv4 must be rejected
	}

	for _, tc := range tests {
		t.Run(tc.ip, func(t *testing.T) {
			got, err := ipv6PTRName(tc.ip)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", tc.ip, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

// ── ptrNameForIP ──────────────────────────────────────────────────────────────

func TestPtrNameForIP(t *testing.T) {
	got, err := ptrNameForIP("192.0.2.1", "A")
	if err != nil || got != "1.2.0.192.in-addr.arpa." {
		t.Fatalf("A: want 1.2.0.192.in-addr.arpa., got %q err %v", got, err)
	}

	got, err = ptrNameForIP("2001:db8::1", "AAAA")
	if err != nil {
		t.Fatalf("AAAA: unexpected error: %v", err)
	}

	if got == "" {
		t.Fatal("AAAA: expected non-empty PTR name")
	}

	_, err = ptrNameForIP("192.0.2.1", "MX")
	if err == nil {
		t.Fatal("MX: expected error for unsupported type")
	}
}

// ── findBestReverseZoneFromList ───────────────────────────────────────────────

func TestFindBestReverseZoneFromList(t *testing.T) {
	zones := []string{
		"in-addr.arpa.",
		"192.in-addr.arpa.",
		"2.0.192.in-addr.arpa.",
	}

	tests := []struct {
		ptrName string
		want    string
	}{
		// Exact match to most-specific zone
		{"1.2.0.192.in-addr.arpa.", "2.0.192.in-addr.arpa."},
		// Matches /8 zone only
		{"1.0.0.10.in-addr.arpa.", "in-addr.arpa."},
		// Matches /8 zone (192.) but not /24
		{"1.0.168.192.in-addr.arpa.", "192.in-addr.arpa."},
		// No match
		{"1.2.3.4.in-addr.arpa.", "in-addr.arpa."},
		// PTR name equals zone apex
		{"2.0.192.in-addr.arpa.", "2.0.192.in-addr.arpa."},
	}

	for _, tc := range tests {
		t.Run(tc.ptrName, func(t *testing.T) {
			got := findBestReverseZoneFromList(tc.ptrName, zones)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFindBestReverseZoneFromList_NoMatch(t *testing.T) {
	zones := []string{"2.0.192.in-addr.arpa."}

	got := findBestReverseZoneFromList("1.2.3.4.in-addr.arpa.", zones)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFindBestReverseZoneFromList_EmptyList(t *testing.T) {
	got := findBestReverseZoneFromList("1.2.0.192.in-addr.arpa.", nil)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// ── ipsFromCurrentZone ────────────────────────────────────────────────────────

func TestIPsFromCurrentZone(t *testing.T) {
	zone := &pdnsapi.Zone{
		RRsets: []pdnsapi.RRset{
			{
				Name: strPtr("www.example.com."),
				Type: rrType("A"),
				Records: []pdnsapi.Record{
					{Content: strPtr("192.0.2.1")},
					{Content: strPtr("192.0.2.2")},
				},
			},
			{
				Name:    strPtr("www.example.com."),
				Type:    rrType("AAAA"),
				Records: []pdnsapi.Record{{Content: strPtr("2001:db8::1")}},
			},
		},
	}

	t.Run("existing A record", func(t *testing.T) {
		ips := ipsFromCurrentZone(zone, "www.example.com.", "A")
		if len(ips) != 2 {
			t.Fatalf("want 2 IPs, got %d", len(ips))
		}
	})

	t.Run("existing AAAA record", func(t *testing.T) {
		ips := ipsFromCurrentZone(zone, "www.example.com.", "AAAA")
		if len(ips) != 1 || ips[0] != "2001:db8::1" {
			t.Fatalf("unexpected IPs: %v", ips)
		}
	})

	t.Run("non-existent record", func(t *testing.T) {
		ips := ipsFromCurrentZone(zone, "mail.example.com.", "A")
		if len(ips) != 0 {
			t.Fatalf("expected empty, got %v", ips)
		}
	})

	t.Run("nil zone", func(t *testing.T) {
		ips := ipsFromCurrentZone(nil, "www.example.com.", "A")
		if len(ips) != 0 {
			t.Fatalf("expected empty for nil zone, got %v", ips)
		}
	})

	t.Run("fqdn without trailing dot normalised", func(t *testing.T) {
		ips := ipsFromCurrentZone(zone, "www.example.com", "A") // no trailing dot
		if len(ips) != 2 {
			t.Fatalf("want 2 IPs, got %d", len(ips))
		}
	})
}

// ── ptrContentFromZone ────────────────────────────────────────────────────────

func TestPTRContentFromZone(t *testing.T) {
	zone := &pdnsapi.Zone{
		RRsets: []pdnsapi.RRset{
			{
				Name:    strPtr("1.2.0.192.in-addr.arpa."),
				Type:    rrType("PTR"),
				Records: []pdnsapi.Record{{Content: strPtr("www.example.com.")}},
			},
		},
	}

	t.Run("existing PTR", func(t *testing.T) {
		got := ptrContentFromZone(zone, "1.2.0.192.in-addr.arpa.")
		if got != "www.example.com." {
			t.Fatalf("want www.example.com., got %q", got)
		}
	})

	t.Run("non-existent PTR", func(t *testing.T) {
		got := ptrContentFromZone(zone, "2.2.0.192.in-addr.arpa.")
		if got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("nil zone", func(t *testing.T) {
		got := ptrContentFromZone(nil, "1.2.0.192.in-addr.arpa.")
		if got != "" {
			t.Fatalf("expected empty for nil zone, got %q", got)
		}
	})

	t.Run("case-insensitive lookup", func(t *testing.T) {
		got := ptrContentFromZone(zone, "1.2.0.192.IN-ADDR.ARPA.")
		if got != "www.example.com." {
			t.Fatalf("case-insensitive lookup failed, got %q", got)
		}
	})
}

// ── loadZoneSettings / saveZoneSettings ───────────────────────────────────────

func TestZoneSettings_DefaultsWhenMissing(t *testing.T) {
	db := newTestDB(t)

	s := loadZoneSettings(db, "example.com.")
	if s.AutoPTR {
		t.Fatal("expected AutoPTR=false when no settings exist")
	}
}

func TestZoneSettings_SaveAndLoad(t *testing.T) {
	db := newTestDB(t)

	if err := saveZoneSettings(db, "example.com.", ZoneSettings{AutoPTR: true}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := loadZoneSettings(db, "example.com.")
	if !got.AutoPTR {
		t.Fatal("expected AutoPTR=true after save")
	}
}

func TestZoneSettings_MultipleZonesIsolated(t *testing.T) {
	db := newTestDB(t)

	if err := saveZoneSettings(db, "example.com.", ZoneSettings{AutoPTR: true}); err != nil {
		t.Fatalf("save example.com.: %v", err)
	}

	if err := saveZoneSettings(db, "other.com.", ZoneSettings{AutoPTR: false}); err != nil {
		t.Fatalf("save other.com.: %v", err)
	}

	if s := loadZoneSettings(db, "example.com."); !s.AutoPTR {
		t.Fatal("example.com. AutoPTR should be true")
	}

	if s := loadZoneSettings(db, "other.com."); s.AutoPTR {
		t.Fatal("other.com. AutoPTR should be false")
	}
}

func TestZoneSettings_UpdateExisting(t *testing.T) {
	db := newTestDB(t)

	if err := saveZoneSettings(db, "example.com.", ZoneSettings{AutoPTR: true}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	if err := saveZoneSettings(db, "example.com.", ZoneSettings{AutoPTR: false}); err != nil {
		t.Fatalf("second save: %v", err)
	}

	got := loadZoneSettings(db, "example.com.")
	if got.AutoPTR {
		t.Fatal("expected AutoPTR=false after update")
	}
}

func TestZoneSettings_NilDBReturnsDefaults(t *testing.T) {
	s := loadZoneSettings(nil, "example.com.")

	if s.AutoPTR {
		t.Fatal("nil DB should return zero-value settings")
	}
}
