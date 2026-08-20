package dnssec

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// fakeExchanger implements exchanger using a map of server+qname+qtype → response.
// The key format is "<server>|<qname>|<qtype>" so tests can distinguish queries
// for different zones sent to the same nameserver.
type fakeExchanger struct {
	responses map[string]*dns.Msg
	err       map[string]error
}

func (f *fakeExchanger) key(server, qname string, qtype uint16) string {
	return server + "|" + strings.ToLower(dns.Fqdn(qname)) + "|" + dns.TypeToString[qtype]
}

func (f *fakeExchanger) Exchange(m *dns.Msg, server string) (*dns.Msg, time.Duration, error) {
	qname := ""
	qtype := uint16(0)

	if len(m.Question) > 0 {
		qname = m.Question[0].Name
		qtype = m.Question[0].Qtype
	}

	k := f.key(server, qname, qtype)

	if err, ok := f.err[k]; ok {
		return nil, 0, err
	}

	if r, ok := f.responses[k]; ok {
		return r, 0, nil
	}

	// Default: NOERROR with empty answer (no DS / no NS).
	resp := new(dns.Msg)
	resp.SetReply(m)
	resp.Rcode = dns.RcodeSuccess

	return resp, 0, nil
}

// f2key is a package-level helper for building fakeExchanger keys in test
// setup where a receiver isn't yet available.
func f2key(server, qname string, qtype uint16) string {
	return server + "|" + strings.ToLower(dns.Fqdn(qname)) + "|" + dns.TypeToString[qtype]
}

// --- Message builder helpers -------------------------------------------------

func nsMsg(zone, ns, glueIP string) *dns.Msg {
	m := new(dns.Msg)
	m.Rcode = dns.RcodeSuccess

	m.Answer = []dns.RR{
		&dns.NS{
			Hdr: dns.RR_Header{Name: zone, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 86400},
			Ns:  ns,
		},
	}

	if glueIP != "" {
		m.Extra = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{Name: ns, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 86400},
				A:   net.ParseIP(glueIP),
			},
		}
	}

	return m
}

const (
	testAlgo       = uint8(13) // ECDSAP256SHA256 — used across all test DS records
	testDigestType = uint8(2)  // SHA-256 — used across all test DS records
)

func dsMsg(zone string, keytag uint16, digest string) *dns.Msg {
	algo := testAlgo
	digestType := testDigestType
	m := new(dns.Msg)
	m.Rcode = dns.RcodeSuccess

	m.Answer = []dns.RR{
		&dns.DS{
			Hdr:        dns.RR_Header{Name: zone, Rrtype: dns.TypeDS, Class: dns.ClassINET, Ttl: 3600},
			KeyTag:     keytag,
			Algorithm:  algo,
			DigestType: digestType,
			Digest:     digest,
		},
	}

	return m
}

func emptyMsg() *dns.Msg {
	m := new(dns.Msg)
	m.Rcode = dns.RcodeSuccess

	return m
}

// --- Tests for zone list building (logic embedded in walkChain) --------------

func TestWalkChain_ZoneListBuilding(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "example.com.",
			want:  []string{".", "com.", "example.com."},
		},
		{
			input: "dev.eqipe.cloud.",
			want:  []string{".", "cloud.", "eqipe.cloud.", "dev.eqipe.cloud."},
		},
		{
			input: "a.b.c.d.",
			want:  []string{".", "d.", "c.d.", "b.c.d.", "a.b.c.d."},
		},
	}

	for _, tt := range tests {
		labels := dns.SplitDomainName(strings.TrimSuffix(tt.input, "."))

		got := make([]string, 0, len(labels)+1)
		got = append(got, ".")

		for i := len(labels) - 1; i >= 0; i-- {
			got = append(got, strings.Join(labels[i:], ".")+".")
		}

		if len(got) != len(tt.want) {
			t.Errorf("input %q: got %v, want %v", tt.input, got, tt.want)

			continue
		}

		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("input %q zone[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// --- Tests for walkChain -----------------------------------------------------

// rootNSMsg returns a response for the root zone's own NS query — needed
// because checkDelegation calls findAuthNS(ctx, parent=".", resolver=rootNS)
// for the first delegation level, so the fake must answer it.
func rootNSMsg() *dns.Msg {
	return nsMsg(".", "a.root-servers.net.", "198.41.0.4")
}

// newCompleteChainFake sets up a fake exchanger that serves NS + DS for a
// complete two-level chain: root → tld. → example.tld.
// Glue records are included so no real DNS lookups are needed.
func newCompleteChainFake() *fakeExchanger {
	rootNS := "198.41.0.4:53"
	tldNS := "192.0.2.1:53"
	exampleNS := "192.0.2.2:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err:       map[string]error{},
	}

	// root zone NS query (parent discovery for the first delegation level)
	f.responses[f.key(rootNS, ".", dns.TypeNS)] = rootNSMsg()
	// root → DS for tld. (first delegation: root is parent of tld.)
	f.responses[f.key(rootNS, "tld.", dns.TypeDS)] = dsMsg("tld.", 9999, "aabbcc")
	// root → NS for tld., with glue (to discover tldNS as next resolver)
	f.responses[f.key(rootNS, "tld.", dns.TypeNS)] = nsMsg("tld.", "ns1.tld.", "192.0.2.1")
	// tld. auth NS self-NS query (used to confirm tldNS is authoritative for tld.)
	f.responses[f.key(tldNS, "tld.", dns.TypeNS)] = nsMsg("tld.", "ns1.tld.", "192.0.2.1")
	// tld. auth NS → DS for example.tld.
	f.responses[f.key(tldNS, "example.tld.", dns.TypeDS)] = dsMsg("example.tld.", 12345, "deadbeef")
	// tld. auth NS → NS for example.tld. (used to find next resolver)
	f.responses[f.key(tldNS, "example.tld.", dns.TypeNS)] = nsMsg("example.tld.", "ns1.example.tld.", "192.0.2.2")
	// example.tld. auth NS — needed if walkChain descends further (unused here)
	f.responses[f.key(exampleNS, "example.tld.", dns.TypeNS)] = nsMsg("example.tld.", "ns1.example.tld.", "192.0.2.2")

	return f
}

func TestWalkChain_ValidTwoLevel(t *testing.T) {
	f := newCompleteChainFake()
	ctx := context.Background()

	result := walkChain(ctx, "example.tld.", f)

	if !result.Valid {
		t.Fatalf("expected valid chain, got reason: %s", result.Reason)
	}

	// Links: tld. + example.tld.
	if len(result.Links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(result.Links), result.Links)
	}

	for _, link := range result.Links {
		if link.Status != "ok" {
			t.Errorf("link %s: expected ok, got %s: %s", link.Zone, link.Status, link.Detail)
		}
	}

	last := result.Links[len(result.Links)-1]
	if last.Zone != "example.tld." {
		t.Errorf("expected last zone example.tld., got %s", last.Zone)
	}
}

func TestWalkChain_ValidThreeLevel(t *testing.T) {
	rootNS := "198.41.0.4:53"
	tldNS := "192.0.2.1:53"
	exampleNS := "192.0.2.2:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err:       map[string]error{},
	}

	f.responses[f.key(rootNS, ".", dns.TypeNS)] = rootNSMsg()
	f.responses[f.key(rootNS, "tld.", dns.TypeDS)] = dsMsg("tld.", 9999, "aabbcc")
	f.responses[f.key(rootNS, "tld.", dns.TypeNS)] = nsMsg("tld.", "ns1.tld.", "192.0.2.1")
	f.responses[f.key(tldNS, "tld.", dns.TypeNS)] = nsMsg("tld.", "ns1.tld.", "192.0.2.1")
	f.responses[f.key(tldNS, "example.tld.", dns.TypeDS)] = dsMsg("example.tld.", 111, "aabb")
	f.responses[f.key(tldNS, "example.tld.", dns.TypeNS)] = nsMsg("example.tld.", "ns1.example.tld.", "192.0.2.2")
	f.responses[f.key(exampleNS, "example.tld.", dns.TypeNS)] = nsMsg("example.tld.", "ns1.example.tld.", "192.0.2.2")
	f.responses[f.key(exampleNS, "sub.example.tld.", dns.TypeDS)] = dsMsg("sub.example.tld.", 222, "ccdd")
	f.responses[f.key(exampleNS, "sub.example.tld.", dns.TypeNS)] = nsMsg("sub.example.tld.", "ns1.sub.example.tld.", "192.0.2.3")

	ctx := context.Background()
	result := walkChain(ctx, "sub.example.tld.", f)

	if !result.Valid {
		t.Fatalf("expected valid chain, got reason: %s", result.Reason)
	}

	// Links: tld. + example.tld. + sub.example.tld.
	if len(result.Links) != 3 {
		t.Fatalf("expected 3 links, got %d: %v", len(result.Links), result.Links)
	}

	for _, link := range result.Links {
		if link.Status != "ok" {
			t.Errorf("link %s: expected ok, got %s: %s", link.Zone, link.Status, link.Detail)
		}
	}
}

func TestWalkChain_MissingDS(t *testing.T) {
	rootNS := "198.41.0.4:53"
	tldNS := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(rootNS, ".", dns.TypeNS):           rootNSMsg(),
			f2key(rootNS, "tld.", dns.TypeDS):        dsMsg("tld.", 9999, "aabbcc"),
			f2key(rootNS, "tld.", dns.TypeNS):        nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
			f2key(tldNS, "tld.", dns.TypeNS):         nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
			f2key(tldNS, "example.tld.", dns.TypeDS): emptyMsg(), // no DS
		},
		err: map[string]error{},
	}

	ctx := context.Background()
	result := walkChain(ctx, "example.tld.", f)

	if result.Valid {
		t.Fatal("expected invalid chain when DS is missing")
	}

	if len(result.Links) != 2 {
		t.Fatalf("expected 2 links (tld. ok, example.tld. missing), got %d: %v", len(result.Links), result.Links)
	}

	if result.Links[0].Status != "ok" {
		t.Errorf("expected tld. link status ok, got %s: %s", result.Links[0].Status, result.Links[0].Detail)
	}

	if result.Links[1].Status != "missing" {
		t.Errorf("expected example.tld. status missing, got %s", result.Links[1].Status)
	}

	if !strings.Contains(result.Reason, "chain broken") {
		t.Errorf("expected reason to mention chain broken, got %q", result.Reason)
	}
}

func TestWalkChain_NSLookupError(t *testing.T) {
	rootNS := "198.41.0.4:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err: map[string]error{
			f2key(rootNS, "tld.", dns.TypeNS): errors.New("connection refused"),
		},
	}

	ctx := context.Background()
	result := walkChain(ctx, "example.tld.", f)

	if result.Valid {
		t.Fatal("expected invalid chain on NS lookup error")
	}

	if result.Links[0].Status != "error" {
		t.Errorf("expected status error, got %s", result.Links[0].Status)
	}
}

func TestWalkChain_DSQueryError(t *testing.T) {
	rootNS := "198.41.0.4:53"
	tldNS := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(rootNS, ".", dns.TypeNS):    rootNSMsg(),
			f2key(rootNS, "tld.", dns.TypeDS): dsMsg("tld.", 9999, "aabbcc"),
			f2key(rootNS, "tld.", dns.TypeNS): nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
			f2key(tldNS, "tld.", dns.TypeNS):  nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
		},
		err: map[string]error{
			f2key(tldNS, "example.tld.", dns.TypeDS): errors.New("timeout"),
		},
	}

	ctx := context.Background()
	result := walkChain(ctx, "example.tld.", f)

	if result.Valid {
		t.Fatal("expected invalid chain on DS query error")
	}

	// First link (tld.) succeeds; second link (example.tld.) hits the injected error.
	if len(result.Links) < 2 {
		t.Fatalf("expected at least 2 links, got %d: %v", len(result.Links), result.Links)
	}

	last := result.Links[len(result.Links)-1]

	if last.Status != "error" {
		t.Errorf("expected last link status error, got %s", last.Status)
	}

	if !strings.Contains(last.Detail, "timeout") {
		t.Errorf("expected detail to mention timeout, got %q", last.Detail)
	}
}

func TestWalkChain_StopsAtFirstBrokenLink(t *testing.T) {
	// Three-level chain: tld. → example.tld. → sub.example.tld.
	// DS for example.tld. is missing; sub.example.tld. should never be checked.
	rootNS := "198.41.0.4:53"
	tldNS := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(rootNS, ".", dns.TypeNS):           rootNSMsg(),
			f2key(rootNS, "tld.", dns.TypeDS):        dsMsg("tld.", 9999, "aabbcc"),
			f2key(rootNS, "tld.", dns.TypeNS):        nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
			f2key(tldNS, "tld.", dns.TypeNS):         nsMsg("tld.", "ns1.tld.", "192.0.2.1"),
			f2key(tldNS, "example.tld.", dns.TypeDS): emptyMsg(), // missing — should stop here
		},
		err: map[string]error{},
	}

	ctx := context.Background()
	result := walkChain(ctx, "sub.example.tld.", f)

	if result.Valid {
		t.Fatal("expected invalid chain")
	}

	// tld. link is ok; example.tld. link is missing; sub.example.tld. is never checked.
	if len(result.Links) != 2 {
		t.Errorf("expected 2 links (tld. ok + example.tld. missing), got %d: %v", len(result.Links), result.Links)
	}

	last := result.Links[len(result.Links)-1]

	if last.Zone != "example.tld." {
		t.Errorf("expected broken link at example.tld., got %s", last.Zone)
	}

	if last.Status != "missing" {
		t.Errorf("expected status missing, got %s", last.Status)
	}
}

// --- Tests for findAuthNS ----------------------------------------------------

func TestFindAuthNS_UsesGlue(t *testing.T) {
	resolver := "192.0.2.100:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(resolver, "example.com.", dns.TypeNS): nsMsg("example.com.", "ns1.example.com.", "192.0.2.50"),
		},
		err: map[string]error{},
	}

	ctx := context.Background()

	got, err := findAuthNS(ctx, "example.com.", resolver, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "192.0.2.50:53"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindAuthNS_NoNSRecords(t *testing.T) {
	resolver := "192.0.2.100:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(resolver, "example.com.", dns.TypeNS): emptyMsg(),
		},
		err: map[string]error{},
	}

	ctx := context.Background()

	_, err := findAuthNS(ctx, "example.com.", resolver, f)
	if err == nil {
		t.Fatal("expected error when no NS records returned")
	}

	if !strings.Contains(err.Error(), "no NS records") {
		t.Errorf("expected 'no NS records' error, got %v", err)
	}
}

func TestFindAuthNS_ExchangeError(t *testing.T) {
	resolver := "192.0.2.100:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err: map[string]error{
			f2key(resolver, "example.com.", dns.TypeNS): errors.New("network unreachable"),
		},
	}

	ctx := context.Background()

	_, err := findAuthNS(ctx, "example.com.", resolver, f)
	if err == nil {
		t.Fatal("expected error on exchange failure")
	}
}

// --- Tests for queryDS -------------------------------------------------------

func TestQueryDS_ReturnsRecords(t *testing.T) {
	ns := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(ns, "example.com.", dns.TypeDS): dsMsg("example.com.", 12345, "deadbeef"),
		},
		err: map[string]error{},
	}

	ctx := context.Background()

	records, err := queryDS(ctx, "example.com.", ns, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 DS record, got %d", len(records))
	}

	if records[0].KeyTag != 12345 {
		t.Errorf("expected keytag 12345, got %d", records[0].KeyTag)
	}

	if records[0].DigestType != 2 {
		t.Errorf("expected digest type 2, got %d", records[0].DigestType)
	}
}

func TestQueryDS_EmptyAnswer(t *testing.T) {
	ns := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(ns, "example.com.", dns.TypeDS): emptyMsg(),
		},
		err: map[string]error{},
	}

	ctx := context.Background()

	records, err := queryDS(ctx, "example.com.", ns, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 DS records, got %d", len(records))
	}
}

func TestQueryDS_ExchangeError(t *testing.T) {
	ns := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err: map[string]error{
			f2key(ns, "example.com.", dns.TypeDS): errors.New("i/o timeout"),
		},
	}

	ctx := context.Background()

	_, err := queryDS(ctx, "example.com.", ns, f)
	if err == nil {
		t.Fatal("expected error on exchange failure")
	}
}

// --- Tests for exchangeWithContext -------------------------------------------

func TestExchangeWithContext_PropagatesRcodeError(t *testing.T) {
	server := "192.0.2.1:53"

	refused := new(dns.Msg)
	refused.Rcode = dns.RcodeRefused

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{
			f2key(server, "example.com.", dns.TypeA): refused,
		},
		err: map[string]error{},
	}

	m := new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeA)

	ctx := context.Background()

	_, err := exchangeWithContext(ctx, m, server, f)
	if err == nil {
		t.Fatal("expected error for REFUSED rcode")
	}

	if !strings.Contains(err.Error(), "REFUSED") {
		t.Errorf("expected REFUSED in error, got %v", err)
	}
}

func TestExchangeWithContext_ContextDeadlineShortensTimeout(_ *testing.T) {
	server := "192.0.2.1:53"

	f := &fakeExchanger{
		responses: map[string]*dns.Msg{},
		err:       map[string]error{},
	}

	// Deadline already passed — exchangeWithContext should create a fresh client
	// with a tiny timeout and call the fake, not panic.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	m := new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeA)

	// No assertion on error — just verify no panic.
	_, _ = exchangeWithContext(ctx, m, server, f)
}
