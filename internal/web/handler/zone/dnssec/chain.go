package dnssec

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
)

const (
	rootResolver    = "198.41.0.4:53" // a.root-servers.net
	chainTimeout    = 5 * time.Second
	chainHTTPTimeout = 60 * time.Second
	edns0BufSize    = 4096
)

// exchanger is the interface used to send a DNS message and receive a reply.
// The production implementation uses dns.Client; tests inject a fake.
type exchanger interface {
	Exchange(m *dns.Msg, server string) (*dns.Msg, time.Duration, error)
}

// realExchanger wraps dns.Client to implement exchanger.
type realExchanger struct{ c *dns.Client }

func (r *realExchanger) Exchange(m *dns.Msg, server string) (*dns.Msg, time.Duration, error) {
	return r.c.Exchange(m, server)
}

// ChainLink represents one delegation level in the DNSSEC trust chain.
type ChainLink struct {
	Zone   string `json:"zone"`
	Status string `json:"status"` // "ok", "missing", "error"
	Detail string `json:"detail"`
}

// ChainResult is the JSON response for the trust chain check endpoint.
type ChainResult struct {
	Valid  bool        `json:"valid"`
	Links  []ChainLink `json:"links"`
	Reason string      `json:"reason,omitempty"`
}

// CheckChain performs a full DNSSEC trust chain walk from the DNS root down to
// the requested zone and returns the result as JSON.
// Handles GET /zone/edit/:name/dnssec/chain.
func (s *Service) CheckChain(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), chainHTTPTimeout)
	defer cancel()

	ex := &realExchanger{c: &dns.Client{Timeout: chainTimeout}}
	result := walkChain(ctx, zoneName, ex)

	log.Info().
		Str("zone", zoneName).
		Bool("valid", result.Valid).
		Int("links", len(result.Links)).
		Msg("DNSSEC chain check completed")

	return c.JSON(fiber.Map{"success": true, "chain": result})
}

// walkChain walks from the root down to zoneName, verifying that each
// delegation has a DS record published at the parent.
func walkChain(ctx context.Context, zoneName string, ex exchanger) ChainResult {
	// Build the list of zones to verify from root down.
	// e.g. "dev.eqipe.cloud." → [".", "cloud.", "eqipe.cloud.", "dev.eqipe.cloud."]
	labels := dns.SplitDomainName(strings.TrimSuffix(zoneName, "."))

	zones := make([]string, 0, len(labels)+1)
	zones = append(zones, ".")

	for i := len(labels) - 1; i >= 0; i-- {
		zones = append(zones, strings.Join(labels[i:], ".")+".")
	}

	var links []ChainLink

	// Walk each delegation level: check that the child zone has a DS record
	// in the parent, starting from the root.
	// The root itself is the trust anchor — we skip the DS check for ".".
	resolver := rootResolver

	for i := 1; i < len(zones); i++ {
		child := zones[i]
		parent := zones[i-1]

		link, nextResolver := checkDelegation(ctx, child, parent, resolver, ex)
		links = append(links, link)

		if link.Status != "ok" {
			return ChainResult{
				Valid:  false,
				Links:  links,
				Reason: fmt.Sprintf("chain broken at %s: %s", child, link.Detail),
			}
		}

		// Advance the resolver to the child zone's authoritative nameserver
		// for the next iteration.
		if nextResolver != "" {
			resolver = nextResolver
		}
	}

	return ChainResult{Valid: true, Links: links}
}

// checkDelegation verifies that a DS record for child exists at the parent,
// querying through the given resolver (an authoritative NS for the parent).
// Returns the ChainLink result and the IP of an authoritative NS for the child
// zone (to use as resolver for the next level).
func checkDelegation(ctx context.Context, child, parent, resolver string, ex exchanger) (ChainLink, string) {
	link := ChainLink{Zone: child}

	// Step 1: find an authoritative NS for the parent zone.
	authNS, err := findAuthNS(ctx, parent, resolver, ex)
	if err != nil {
		link.Status = "error"
		link.Detail = fmt.Sprintf("could not find authoritative NS for %s: %v", parent, err)

		return link, ""
	}

	// Step 2: query that NS directly for the child DS record.
	ds, err := queryDS(ctx, child, authNS, ex)
	if err != nil {
		link.Status = "error"
		link.Detail = fmt.Sprintf("DS query to %s for %s failed: %v", authNS, child, err)

		return link, ""
	}

	if len(ds) == 0 {
		link.Status = "missing"
		link.Detail = fmt.Sprintf("no DS record for %s found at parent %s (queried %s)", child, parent, authNS)

		return link, ""
	}

	// Format DS records for display.
	var dsStrings []string

	for _, r := range ds {
		dsStrings = append(dsStrings, r.String())
	}

	link.Status = "ok"
	link.Detail = strings.Join(dsStrings, " | ")

	// Step 3: find an authoritative NS for the child zone to use next.
	// Ignore the error — if we cannot resolve the child NS we simply reuse
	// the current resolver for the next level rather than failing the chain.
	// Best-effort: if child NS cannot be resolved, reuse the current resolver.
	childNS, _ := findAuthNS(ctx, child, authNS, ex) //nolint:errcheck // non-fatal; caller falls back to parent resolver

	return link, childNS
}

// findAuthNS resolves a nameserver IP for zone by querying resolver.
// It follows NS → A to get a usable IP:port.
func findAuthNS(ctx context.Context, zone, resolver string, ex exchanger) (string, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(zone), dns.TypeNS)
	m.RecursionDesired = false

	r, err := exchangeWithContext(ctx, m, resolver, ex)
	if err != nil {
		return "", err
	}

	// Look for NS records in the answer or authority section.
	var nsNames []string

	for _, rr := range append(r.Answer, r.Ns...) {
		if ns, ok := rr.(*dns.NS); ok {
			nsNames = append(nsNames, ns.Ns)
		}
	}

	// Also check glue records in the additional section.
	glue := map[string]string{}

	for _, rr := range r.Extra {
		if a, ok := rr.(*dns.A); ok {
			glue[strings.ToLower(a.Hdr.Name)] = a.A.String()
		}
	}

	for _, ns := range nsNames {
		if ip, ok := glue[strings.ToLower(ns)]; ok {
			return net.JoinHostPort(ip, "53"), nil
		}

		// No glue — resolve the NS host separately via the system resolver.
		ips, err := net.DefaultResolver.LookupHost(ctx, ns)
		if err == nil && len(ips) > 0 {
			return net.JoinHostPort(ips[0], "53"), nil
		}
	}

	if len(nsNames) == 0 {
		return "", fmt.Errorf("no NS records found for %s", zone)
	}

	return "", fmt.Errorf("could not resolve IP for any NS of %s", zone)
}

// queryDS queries the given nameserver for the DS records of zone.
func queryDS(ctx context.Context, zone, ns string, ex exchanger) ([]*dns.DS, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(zone), dns.TypeDS)
	m.RecursionDesired = false
	m.SetEdns0(edns0BufSize, true)

	r, err := exchangeWithContext(ctx, m, ns, ex)
	if err != nil {
		return nil, err
	}

	var records []*dns.DS

	for _, rr := range r.Answer {
		if ds, ok := rr.(*dns.DS); ok {
			records = append(records, ds)
		}
	}

	return records, nil
}

// exchangeWithContext performs a DNS exchange with a per-call deadline derived
// from ctx, falling back to chainTimeout if ctx has no deadline.
func exchangeWithContext(ctx context.Context, m *dns.Msg, server string, ex exchanger) (*dns.Msg, error) {
	// Respect the context deadline: if it is tighter than chainTimeout, set
	// a shorter timeout on a fresh client rather than mutating the shared one.
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining < chainTimeout {
			ex = &realExchanger{c: &dns.Client{Timeout: remaining}}
		}
	}

	r, _, err := ex.Exchange(m, server)
	if err != nil {
		return nil, err
	}

	if r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError {
		return nil, fmt.Errorf("DNS error: %s", dns.RcodeToString[r.Rcode])
	}

	return r, nil
}
