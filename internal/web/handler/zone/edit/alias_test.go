package zoneedit

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	pdnsapi "github.com/joeig/go-powerdns/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

func rrSet(name, rrType string, contents ...string) pdnsapi.RRset {
	t := pdnsapi.RRType(rrType)
	records := make([]pdnsapi.Record, 0, len(contents))

	for _, c := range contents {
		content := c
		records = append(records, pdnsapi.Record{Content: &content})
	}

	return pdnsapi.RRset{Name: &name, Type: &t, Records: records}
}

func aliasChange(name string) RecordChange {
	return RecordChange{
		Changed: true,
		Name:    name,
		Type:    aliasRecordType,
		TTL:     3600,
		Records: []Record{{Content: "lb.example.net."}},
	}
}

func stubAliasSupport(t *testing.T, support powerdns.AliasSupport, err error) *int {
	t.Helper()

	calls := 0
	orig := lookupAliasSupport
	lookupAliasSupport = func(context.Context) (powerdns.AliasSupport, error) {
		calls++
		return support, err
	}

	t.Cleanup(func() { lookupAliasSupport = orig })

	return &calls
}

func TestChangesAddAlias(t *testing.T) {
	tests := []struct {
		name    string
		changes []RecordChange
		want    bool
	}{
		{"no changes", nil, false},
		{"new ALIAS", []RecordChange{aliasChange("example.com.")}, true},
		{"unchanged ALIAS", []RecordChange{{Type: aliasRecordType, Records: []Record{{Content: "x."}}}}, false},
		{"deleted ALIAS", []RecordChange{{Existed: true, Changed: true, Type: aliasRecordType}}, false},
		{"other type", []RecordChange{{Changed: true, Type: "A", Records: []Record{{Content: "192.0.2.1"}}}}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := changesAddAlias(tc.changes); got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestAliasConflicts(t *testing.T) {
	tests := []struct {
		name    string
		current []pdnsapi.RRset
		changes []RecordChange
		want    []string
	}{
		{
			name:    "ALIAS on an empty name",
			current: []pdnsapi.RRset{rrSet("example.com.", "MX", "10 mx.example.com.")},
			changes: []RecordChange{aliasChange("example.com.")},
			want:    nil,
		},
		{
			name:    "ALIAS next to existing A",
			current: []pdnsapi.RRset{rrSet("example.com.", "A", "192.0.2.1")},
			changes: []RecordChange{aliasChange("example.com.")},
			want:    []string{"example.com. (A)"},
		},
		{
			name:    "A deleted in the same request",
			current: []pdnsapi.RRset{rrSet("example.com.", "A", "192.0.2.1")},
			changes: []RecordChange{
				{Existed: true, Changed: true, Name: "example.com.", Type: "A"},
				aliasChange("example.com."),
			},
			want: nil,
		},
		{
			name: "A and AAAA, names compared case-insensitively and without trailing dot",
			current: []pdnsapi.RRset{
				rrSet("Example.com.", "A", "192.0.2.1"),
				rrSet("example.com.", "AAAA", "2001:db8::1"),
			},
			changes: []RecordChange{aliasChange("example.com")},
			want:    []string{"example.com. (A)", "example.com. (AAAA)"},
		},
		{
			name:    "CNAME added next to existing ALIAS",
			current: []pdnsapi.RRset{rrSet("www.example.com.", aliasRecordType, "lb.example.net.")},
			changes: []RecordChange{{
				Changed: true, Name: "www.example.com.", Type: "CNAME",
				Records: []Record{{Content: "other.example.net."}},
			}},
			want: []string{"www.example.com. (CNAME)"},
		},
		{
			name:    "A on another name",
			current: []pdnsapi.RRset{rrSet("www.example.com.", "A", "192.0.2.1")},
			changes: []RecordChange{aliasChange("example.com.")},
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := aliasConflicts(tc.current, tc.changes); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestCheckAliasChanges(t *testing.T) {
	ready := powerdns.AliasSupport{ExpandAlias: true, Resolver: "192.0.2.53"}

	t.Run("no ALIAS changes skips the lookup", func(t *testing.T) {
		calls := stubAliasSupport(t, powerdns.AliasSupport{}, nil)

		changes := []RecordChange{{Changed: true, Name: "example.com.", Type: "A", Records: []Record{{Content: "192.0.2.1"}}}}
		if err := checkAliasChanges(t.Context(), nil, changes); err != nil {
			t.Fatalf("want nil, got %v", err)
		}

		if *calls != 0 {
			t.Fatalf("lookup called %d times, want 0", *calls)
		}
	})

	t.Run("server resolves ALIAS", func(t *testing.T) {
		stubAliasSupport(t, ready, nil)

		if err := checkAliasChanges(t.Context(), nil, []RecordChange{aliasChange("example.com.")}); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})

	t.Run("expand-alias disabled", func(t *testing.T) {
		stubAliasSupport(t, powerdns.AliasSupport{Resolver: "192.0.2.53"}, nil)

		err := checkAliasChanges(t.Context(), nil, []RecordChange{aliasChange("example.com.")})
		if err == nil || !strings.Contains(err.Error(), "expand-alias is disabled") {
			t.Fatalf("want expand-alias error, got %v", err)
		}

		if errors.Is(err, errAliasUnverified) {
			t.Fatal("configuration problem must not be reported as unverified")
		}
	})

	t.Run("configuration cannot be read", func(t *testing.T) {
		stubAliasSupport(t, powerdns.AliasSupport{}, errors.New("connection refused"))

		err := checkAliasChanges(t.Context(), nil, []RecordChange{aliasChange("example.com.")})
		if !errors.Is(err, errAliasUnverified) {
			t.Fatalf("want errAliasUnverified, got %v", err)
		}
	})

	t.Run("conflict is reported before the lookup", func(t *testing.T) {
		calls := stubAliasSupport(t, ready, nil)
		current := []pdnsapi.RRset{rrSet("example.com.", "A", "192.0.2.1")}

		err := checkAliasChanges(t.Context(), current, []RecordChange{aliasChange("example.com.")})
		if err == nil || !strings.Contains(err.Error(), "example.com. (A)") {
			t.Fatalf("want conflict error, got %v", err)
		}

		if *calls != 0 {
			t.Fatalf("lookup called %d times, want 0", *calls)
		}
	})
}

func TestAnnotateAliasHelp(t *testing.T) {
	options := func() []RecordTypeOption {
		return []RecordTypeOption{
			{Type: "A", Help: "IPv4"},
			{Type: aliasRecordType, Help: "Alias to another hostname."},
		}
	}

	t.Run("warns when ALIAS is not resolved", func(t *testing.T) {
		stubAliasSupport(t, powerdns.AliasSupport{}, nil)

		got := annotateAliasHelp(t.Context(), options())
		if !strings.HasPrefix(got[1].Help, "Warning: ALIAS records are not resolved") ||
			!strings.HasSuffix(got[1].Help, "Alias to another hostname.") {
			t.Fatalf("unexpected help text %q", got[1].Help)
		}

		if got[0].Help != "IPv4" {
			t.Fatalf("other types must stay unchanged, got %q", got[0].Help)
		}
	})

	t.Run("unchanged when ALIAS is resolved", func(t *testing.T) {
		stubAliasSupport(t, powerdns.AliasSupport{ExpandAlias: true, Resolver: "192.0.2.53"}, nil)

		if got := annotateAliasHelp(t.Context(), options()); got[1].Help != "Alias to another hostname." {
			t.Fatalf("unexpected help text %q", got[1].Help)
		}
	})

	t.Run("unchanged when the configuration cannot be read", func(t *testing.T) {
		stubAliasSupport(t, powerdns.AliasSupport{}, errors.New("timeout"))

		if got := annotateAliasHelp(t.Context(), options()); got[1].Help != "Alias to another hostname." {
			t.Fatalf("unexpected help text %q", got[1].Help)
		}
	})

	t.Run("no lookup without ALIAS", func(t *testing.T) {
		calls := stubAliasSupport(t, powerdns.AliasSupport{}, nil)

		annotateAliasHelp(t.Context(), []RecordTypeOption{{Type: "A"}})

		if *calls != 0 {
			t.Fatalf("lookup called %d times, want 0", *calls)
		}
	})
}
