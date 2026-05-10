package context7

import "testing"

func TestParseKey_Valid(t *testing.T) {
	_, err := ParseKey("ctx7sk_abcdef1234567890wxyz")
	if err != nil {
		t.Error(err)
	}
}

func TestParseKey_MissingPrefix(t *testing.T) {
	_, err := ParseKey("abcdef1234567890wxyz")
	if err == nil {
		t.Error("expected error for missing prefix")
	}
}

func TestParseKey_TooShort(t *testing.T) {
	_, err := ParseKey("ctx7sk_abc")
	if err == nil {
		t.Error("expected error for too short key")
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"ctx7sk_abcdef1234567890wxyz", "ctx7sk_abcd...wxyz"},
		{"ctx7sk_short", "ctx7sk_short"},
	}
	for _, tc := range tests {
		if got := RedactKey(tc.in); got != tc.want {
			t.Errorf("RedactKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}