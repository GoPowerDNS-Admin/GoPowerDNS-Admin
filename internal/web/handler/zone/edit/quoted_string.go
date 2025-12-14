package zoneedit

import "regexp"

// quotedStringSequenceRE matches one or more RFC-1035-style quoted strings
// separated by whitespace. Each quoted string allows escaping via backslash
// (e.g., \" for a literal quote).
var quotedStringSequenceRE = regexp.MustCompile(`^\s*"([^"\\]|\\.)*"(?:\s+"([^"\\]|\\.)*")*\s*$`)

// isQuotedStringSequence returns true if s consists of one or more
// RFC-1035-style quoted strings separated by whitespace: "..." "..."
// It supports escaping of \" inside a quoted string. Simplified check.
func isQuotedStringSequence(s string) bool { return quotedStringSequenceRE.MatchString(s) }
