package zoneedit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	pdnsapi "github.com/joeig/go-powerdns/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

const (
	accessTestForwardZone = "a.example."
	accessTestReverseZone = "1.168.192.in-addr.arpa."
	accessTestIP          = "192.168.1.10"
	accessTestFQDN        = "www.a.example."
)

// newFakeAutoPTRPowerDNS serves one forward and one reverse zone. The reverse
// zone holds a PTR for accessTestIP pointing to accessTestFQDN. It returns a
// counter of PATCH requests sent to the reverse zone.
func newFakeAutoPTRPowerDNS(t *testing.T) *atomic.Int32 {
	t.Helper()

	var patches atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers/localhost/zones":
			_, _ = io.WriteString(w, `[
				{"id": "a.example.", "name": "a.example.", "kind": "Native"},
				{"id": "1.168.192.in-addr.arpa.", "name": "1.168.192.in-addr.arpa.", "kind": "Native"}
			]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers/localhost/zones/"+accessTestReverseZone:
			_, _ = io.WriteString(w, `{
				"id": "1.168.192.in-addr.arpa.", "name": "1.168.192.in-addr.arpa.", "kind": "Native",
				"rrsets": [{"name": "10.1.168.192.in-addr.arpa.", "type": "PTR", "ttl": 300,
					"records": [{"content": "www.a.example.", "disabled": false}]}]
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/servers/localhost/zones/"+accessTestReverseZone:
			patches.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error": "not found"}`)
		}
	}))

	powerdns.Engine.Client = pdnsapi.New(srv.URL, "localhost")

	t.Cleanup(func() {
		powerdns.Engine.Client = nil

		srv.Close()
	})

	return &patches
}

func newAutoPTRTestService(t *testing.T) *Service {
	t.Helper()

	db := newTestDB(t)
	if err := db.AutoMigrate(&models.ActivityLog{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	return &Service{db: db}
}

func forwardZoneWithA(ips ...string) *pdnsapi.Zone {
	records := make([]pdnsapi.Record, 0, len(ips))
	for _, ip := range ips {
		records = append(records, pdnsapi.Record{Content: strPtr(ip)})
	}

	return &pdnsapi.Zone{
		Name:   strPtr(accessTestForwardZone),
		RRsets: []pdnsapi.RRset{{Name: strPtr(accessTestFQDN), Type: rrType("A"), Records: records}},
	}
}

func onlyZone(allowed string) func(string) bool {
	return func(zoneName string) bool { return zoneName == allowed }
}

func TestApplyAutoPTR_CreateSkipsReverseZoneWithoutAccess(t *testing.T) {
	patches := newFakeAutoPTRPowerDNS(t)
	svc := newAutoPTRTestService(t)

	changes := []RecordChange{{
		Changed: true, Name: accessTestFQDN, Type: "A", TTL: 300,
		Records: []Record{{Content: accessTestIP}},
	}}

	result := svc.applyAutoPTR(context.Background(), forwardZoneWithA(), changes,
		onlyZone(accessTestForwardZone), nil, "tester", "127.0.0.1")

	if got := patches.Load(); got != 0 {
		t.Fatalf("expected no PTR change in the reverse zone, got %d PATCH requests", got)
	}

	if len(result.NoAccess) != 1 || result.NoAccess[0] != accessTestIP {
		t.Fatalf("expected %s in NoAccess, got %+v", accessTestIP, result)
	}
}

func TestApplyAutoPTR_CreateWritesReverseZoneWithAccess(t *testing.T) {
	patches := newFakeAutoPTRPowerDNS(t)
	svc := newAutoPTRTestService(t)

	changes := []RecordChange{{
		Changed: true, Name: accessTestFQDN, Type: "A", TTL: 300,
		Records: []Record{{Content: accessTestIP}},
	}}

	result := svc.applyAutoPTR(context.Background(), forwardZoneWithA(), changes,
		func(string) bool { return true }, nil, "tester", "127.0.0.1")

	if got := patches.Load(); got != 1 {
		t.Fatalf("expected one PATCH to the reverse zone, got %d", got)
	}

	if len(result.NoAccess) != 0 || len(result.NoReverseZone) != 0 {
		t.Fatalf("expected no skipped IPs, got %+v", result)
	}
}

func TestApplyAutoPTR_DeleteSkipsReverseZoneWithoutAccess(t *testing.T) {
	patches := newFakeAutoPTRPowerDNS(t)
	svc := newAutoPTRTestService(t)

	changes := []RecordChange{{Existed: true, Changed: true, Name: accessTestFQDN, Type: "A", TTL: 300}}

	result := svc.applyAutoPTR(context.Background(), forwardZoneWithA(accessTestIP), changes,
		onlyZone(accessTestForwardZone), nil, "tester", "127.0.0.1")

	if got := patches.Load(); got != 0 {
		t.Fatalf("expected the PTR to stay, got %d PATCH requests", got)
	}

	if len(result.NoAccess) != 1 || result.NoAccess[0] != accessTestIP {
		t.Fatalf("expected %s in NoAccess, got %+v", accessTestIP, result)
	}
}
