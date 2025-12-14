package zoneedit

import "testing"

func TestEnsureQuotedContent_TXT(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{"unquoted simple", `v=spf1 a ~all`, `"v=spf1 a ~all"`},
		{"already quoted", `"hello world"`, `"hello world"`},
		{"multi quoted parts", `"part1" "part2"`, `"part1" "part2"`},
		{"internal quotes", `hello "world"`, `"hello \"world\""`},
		{"empty becomes empty quoted", `  `, `""`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ensureQuotedContent("TXT", tc.in)
			if got != tc.out {
				t.Fatalf("want %q, got %q", tc.out, got)
			}
		})
	}
}

func TestEnsureQuotedContent_SPF(t *testing.T) {
	var (
		got  = ensureQuotedContent("SPF", `v=spf1 a ~all`)
		want = `"v=spf1 a ~all"`
	)

	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestEnsureQuotedContent_OtherTypesUnchanged(t *testing.T) {
	caa := `0 issue "letsencrypt.org"`
	if got := ensureQuotedContent("CAA", caa); got != caa {
		t.Fatalf("CAA should be unchanged: got %q", got)
	}

	naptr := `100 50 "s" "SIP+D2U" "" _sip._udp.example.com.`
	if got := ensureQuotedContent("NAPTR", naptr); got != naptr {
		t.Fatalf("NAPTR should be unchanged: got %q", got)
	}
}

func TestEnsureQuotedContent_URI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{"just url", `https://www.heise.de/`, `0 0 "https://www.heise.de/"`},
		{"quoted url", `"https://www.heise.de/"`, `0 0 "https://www.heise.de/"`},
		{"prio weight url", `10 5 https://www.heise.de/`, `10 5 "https://www.heise.de/"`},
		{"prio weight quoted", `10 5 "https://www.heise.de/"`, `10 5 "https://www.heise.de/"`},
		{"invalid digits default", `x y https://example/`, `0 0 "https://example/"`},
		{"idempotent already normalized", `0 0 "https://www.heise.de/"`, `0 0 "https://www.heise.de/"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ensureQuotedContent("URI", tc.in)
			if got != tc.out {
				t.Fatalf("want %q, got %q", tc.out, got)
			}
		})
	}
}
