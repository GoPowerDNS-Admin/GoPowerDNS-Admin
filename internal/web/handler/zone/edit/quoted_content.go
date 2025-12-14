package zoneedit

import (
	"fmt"
	"strings"
)

// ensureQuotedContent ensures that DNS record content is correctly wrapped in
// double quotes for RR types that require it (TXT, SPF). If the content is
// already a valid sequence of quoted strings, it's returned unchanged. If not,
// embedded quotes are escaped and the whole content is wrapped in quotes.
// For other RR types, content is returned unchanged.
func ensureQuotedContent(rrType, content string) string {
	if content == "" {
		return content
	}

	t := strings.ToUpper(strings.TrimSpace(rrType))
	switch t {
	case "TXT", "SPF":
		s := strings.TrimSpace(content)
		if s == "" {
			return `""`
		}

		if isQuotedStringSequence(s) {
			return s
		}
		// Escape embedded quotes and wrap
		s = strings.ReplaceAll(s, `"`, `\"`)

		return `"` + s + `"`

	case "URI":
		// RFC 7553: priority (uint16) weight (uint16) "target-uri"
		s := strings.TrimSpace(content)
		if s == "" {
			return s
		}

		if m := uriRecordRe.FindStringSubmatch(s); m != nil {
			prio := m[1]
			weight := m[2]
			quotedTarget := m[3]
			unquotedTarget := strings.TrimSpace(m[4])

			if quotedTarget != "" { // already quoted form; keep as is
				return fmt.Sprintf(`%s %s %q`, prio, weight, quotedTarget)
			}

			// Unquoted form: escape embedded quotes and wrap
			if unquotedTarget == "" {
				return fmt.Sprintf(`%s %s ""`, prio, weight)
			}

			unquotedTarget = strings.ReplaceAll(unquotedTarget, `"`, `\"`)

			return fmt.Sprintf(`%s %s %q`, prio, weight, unquotedTarget)
		}

		// Not matching prio/weight form; treat as target only.
		parts := strings.Fields(s)

		target := s
		if len(parts) > 0 {
			target = parts[len(parts)-1]
		}

		if strings.HasPrefix(target, `"`) && strings.HasSuffix(target, `"`) && len(target) >= 2 {
			target = target[1 : len(target)-1]
		}

		target = strings.ReplaceAll(target, `"`, `\"`)

		return fmt.Sprintf(`0 0 %q`, target)
	default:
		return content
	}
}
