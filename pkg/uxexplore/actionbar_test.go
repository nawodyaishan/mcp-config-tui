package uxexplore

import (
	"reflect"
	"testing"
)

func TestParseActionBar_AllScreenVariants(t *testing.T) {
	tests := []struct {
		name string
		view string
		want []string
	}{
		{
			name: "provider ready",
			view: "Provider Readiness\n\n[↑↓] navigate  [v] live validate  [Enter] select  [Esc] back  [?] help  [q] quit",
			want: []string{"?", "down", "enter", "esc", "q", "up", "v"},
		},
		{
			name: "target credentials",
			view: "Select Targets\n\n[↑↓] navigate  [Space] toggle  [i] workspace(off)  [k] add credentials  [Esc] back  [q] quit\nCredentials needed - press [k] to add",
			want: []string{" ", "down", "esc", "i", "k", "q", "up"},
		},
		{
			name: "conflict resolve",
			view: "Resolve Conflict\n\n[s] skip client  [Esc] cancel  [1] use this  [2] use this",
			want: []string{"1", "2", "esc", "s"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseActionBar(tc.view)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
}
