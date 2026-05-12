package tavily

import (
	"testing"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid key", "tvly-abcdef1234567890", false},
		{"valid key with spaces", "  tvly-abcdef1234567890  ", false},
		{"missing prefix", "abcdef1234567890", true},
		{"too short", "tvly-abc", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"valid key", "tvly-abcdef1234567890wxyz", "tvly-abcd...wxyz"},
		{"too short to redact suffix", "tvly-abcdef", "tvly-abcdef"},
		{"exactly minimum length", "tvly-12345678", "tvly-12345678"},
		{"9 char suffix", "tvly-123456789", "tvly-1234...6789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactKey(tt.key); got != tt.want {
				t.Errorf("RedactKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
