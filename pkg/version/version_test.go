package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	s := String()
	if !strings.Contains(s, Version) {
		t.Errorf("expected version in string, got %s", s)
	}
}
