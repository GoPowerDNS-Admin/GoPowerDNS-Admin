package zoneedit

import "testing"

func TestIsQuotedStringSequence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"spaces only", "   \t  ", false},
		{"no quotes", `hello world`, false},
		{"single quoted", `"hello"`, true},
		{"leading and trailing ws", `  "hello"   `, true},
		{"escaped quote inside", `"a\"b"`, true},
		{"escaped backslash inside", `"a\\b"`, true},
		{"two parts space", `"part1" "part2"`, true},
		{"two parts tab", "\"part1\"\t\"part2\"", true},
		{"two parts many spaces", "\"p1\"    \t   \"p2\"", true},
		{"adjacent without space", `"a""b"`, false},
		{"unclosed quote", `"abc`, false},
		{"extra chars between", `"a" x "b"`, false},
		{"single quotes not allowed", `'a'`, false},
		{"newline between parts", `"a"
"b"`, true},
		{"newline around", "\n\t  \"a\"  \n", true},
		{"quoted empty", `""`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isQuotedStringSequence(tc.in)
			if got != tc.want {
				t.Fatalf("input=%q: want %v, got %v", tc.in, tc.want, got)
			}
		})
	}
}
