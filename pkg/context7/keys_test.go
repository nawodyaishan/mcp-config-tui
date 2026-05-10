package context7

import "testing"

func TestParseKey_ValidPrefixes(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "current hyphenated dashboard key",
			key:  "ctx7sk-06801456-a80a-4de8-b6a1-ee189c839918",
			want: "ctx7sk-06801456-a80a-4de8-b6a1-ee189c839918",
		},
		{
			name: "legacy underscore key",
			key:  "ctx7sk_abcdef1234567890wxyz",
			want: "ctx7sk_abcdef1234567890wxyz",
		},
		{
			name: "trims copied whitespace",
			key:  "  ctx7sk-abcdef1234567890wxyz\n",
			want: "ctx7sk-abcdef1234567890wxyz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseKey(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("ParseKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestParseKey_InvalidPrefix(t *testing.T) {
	tests := []string{
		"abcdef1234567890wxyz",
		"ctx7sk/abcdef1234567890wxyz",
		"ctx7skabcdef1234567890wxyz",
		"CTX7SK-abcdef1234567890wxyz",
	}
	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			_, err := ParseKey(key)
			if err == nil {
				t.Fatal("expected error for invalid prefix")
			}
		})
	}
}

func TestParseKey_TooShort(t *testing.T) {
	tests := []string{
		"ctx7sk-abc",
		"ctx7sk_abc",
	}
	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			_, err := ParseKey(key)
			if err == nil {
				t.Fatal("expected error for too short key")
			}
		})
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"ctx7sk-06801456-a80a-4de8-b6a1-ee189c839918", "ctx7sk-0680...9918"},
		{"ctx7sk_abcdef1234567890wxyz", "ctx7sk_abcd...wxyz"},
		{"ctx7sk-short", "ctx7sk-short"},
		{"ctx7sk_short", "ctx7sk_short"},
		{"not-a-context7-key", "not-a-context7-key"},
	}
	for _, tc := range tests {
		if got := RedactKey(tc.in); got != tc.want {
			t.Errorf("RedactKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
