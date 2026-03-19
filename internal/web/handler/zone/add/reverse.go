package zoneadd

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ReverseIPv4Zone converts an IPv4 address or CIDR to a reverse DNS zone name.
// Examples:
//
//	"192.168.1.0/24" → "1.168.192.in-addr.arpa."
//	"10.0.0.0/8"     → "10.in-addr.arpa."
func ReverseIPv4Zone(input string) (string, error) {
	var (
		ip     net.IP
		prefix int
	)

	if strings.Contains(input, "/") {
		_, ipNet, err := net.ParseCIDR(input)
		if err != nil {
			return "", fmt.Errorf("invalid IPv4 CIDR %q: %w", input, err)
		}

		ip = ipNet.IP.To4()
		prefix, _ = ipNet.Mask.Size()
	} else {
		ip = net.ParseIP(input).To4()
		prefix = 32
	}

	if ip == nil {
		return "", fmt.Errorf("not a valid IPv4 address: %q", input)
	}

	octets := max((prefix+7)/8, 1)

	parts := make([]string, octets)
	for i := range octets {
		parts[octets-1-i] = strconv.Itoa(int(ip[i]))
	}

	return strings.Join(parts, ".") + ".in-addr.arpa.", nil
}

const lowerNibbleMask = 0x0F

// ReverseIPv6Zone converts an IPv6 address or CIDR to a reverse DNS zone name.
// Examples:
//
//	"2001:db8::/32"        → "8.b.d.0.1.0.0.2.ip6.arpa."
//	"2a02:d58:2:2000::/64" → "0.0.0.2.8.5.d.0.2.0.a.2.ip6.arpa."
func ReverseIPv6Zone(input string) (string, error) {
	var (
		ip     net.IP
		prefix int
	)

	if strings.Contains(input, "/") {
		_, ipNet, err := net.ParseCIDR(input)
		if err != nil {
			return "", fmt.Errorf("invalid IPv6 CIDR %q: %w", input, err)
		}

		ip = ipNet.IP.To16()
		prefix, _ = ipNet.Mask.Size()
	} else {
		ip = net.ParseIP(input).To16()
		prefix = 128
	}

	if ip == nil {
		return "", fmt.Errorf("not a valid IPv6 address: %q", input)
	}

	nibbleCount := min(max((prefix+3)/4, 1), 32)

	// Extract all 32 nibbles (most significant first)
	nibbles := make([]string, 32)
	for i := range 16 {
		nibbles[i*2] = strconv.FormatUint(uint64(ip[i]>>4), 16)
		nibbles[i*2+1] = strconv.FormatUint(uint64(ip[i]&lowerNibbleMask), 16)
	}

	// Take the first nibbleCount nibbles and reverse them
	selected := nibbles[:nibbleCount]
	reversed := make([]string, nibbleCount)

	for i, n := range selected {
		reversed[nibbleCount-1-i] = n
	}

	return strings.Join(reversed, ".") + ".ip6.arpa.", nil
}
