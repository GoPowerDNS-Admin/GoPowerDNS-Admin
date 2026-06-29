package dashboard

import "testing"

func TestIPQueryToReverseFragment(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantFrag string
		wantIP   bool
	}{
		{"ipv4 full", "192.168.1.50", "50.1.168.192", true},
		{"ipv4 /24 prefix", "192.168.1", "1.168.192", true},
		{"ipv4 /16 prefix", "192.168", "168.192", true},
		{"ipv4 strips leading zeros", "192.168.001.050", "50.1.168.192", true},
		{"ipv4 trailing dot", "192.168.1.", "1.168.192", true},
		{"ipv4 octet out of range", "192.168.1.999", "", false},
		{"ipv4 negative", "192.168.-1", "", false},
		{"hostname not ip", "server01", "", false},
		{"hostname with dots", "host.example.com", "", false},
		{"empty", "", "", false},

		{"ipv6 full", "2001:db8::1", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2", true},
		{"ipv6 prefix two groups", "2001:db8", "8.b.d.0.1.0.0.2", true},
		{"ipv6 prefix pads group", "2001:db8:1", "1.0.0.0.8.b.d.0.1.0.0.2", true},
		{"ipv6 prefix trailing colon", "2001:db8:", "8.b.d.0.1.0.0.2", true},
		{"ipv6 uppercase", "2001:DB8", "8.b.d.0.1.0.0.2", true},
		{"ipv6 bad hex", "2001:zzzz", "", false},
		{"ipv6 group too long", "2001:db8a1", "", false},
		{"bare number is not ipv6", "2001", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frag, isIP := ipQueryToReverseFragment(tt.query)
			if isIP != tt.wantIP {
				t.Fatalf("ipQueryToReverseFragment(%q) isIP = %v, want %v", tt.query, isIP, tt.wantIP)
			}

			if frag != tt.wantFrag {
				t.Errorf("ipQueryToReverseFragment(%q) frag = %q, want %q", tt.query, frag, tt.wantFrag)
			}
		})
	}
}

func TestDotAlignedSuffix(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		suffix string
		want   bool
	}{
		{"equal", "1.168.192", "1.168.192", true},
		{"zone octets end with short prefix", "1.168.192", "192", true},
		{"frag ends with zone octets", "50.1.168.192", "1.168.192", true},
		{"no false partial-octet match", "11.168.192", "1.168.192", false},
		{"not dot aligned mid-octet", "1.168.192", "68.192", false},
		{"empty suffix", "1.168.192", "", false},
		{"not a suffix", "1.168.192", "10.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dotAlignedSuffix(tt.s, tt.suffix); got != tt.want {
				t.Errorf("dotAlignedSuffix(%q, %q) = %v, want %v", tt.s, tt.suffix, got, tt.want)
			}
		})
	}
}
