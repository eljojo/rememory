package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestTruncateHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sha256:abc", "sha256:abc"},                                     // 10 chars
		{"sha256:abcdefghij", "sha256:abcdefghij"},                       // 18 chars
		{"sha256:abcdefghijklm", "sha256:abcdefghijklm"},                 // 21 chars > 20, but expected is first 20
		{"sha256:abcdefghijklmnopqrstuvwxyz", "sha256:abcdefghijklm..."}, // 33 chars -> first 20 + ...
		{"short", "short"},
	}

	for _, tt := range tests {
		result := truncateHash(tt.input)
		if result != tt.expected {
			t.Errorf("truncateHash(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFriendNames(t *testing.T) {
	tests := []struct {
		friends  []project.Friend
		expected string
	}{
		{[]project.Friend{}, ""},
		{[]project.Friend{{Name: "Alice"}}, "Alice"},
		{[]project.Friend{{Name: "Alice"}, {Name: "Bob"}}, "Alice, Bob"},
		{[]project.Friend{{Name: "Alice"}, {Name: "Bob"}, {Name: "Carol"}}, "Alice, Bob, Carol"},
	}

	for _, tt := range tests {
		result := friendNames(tt.friends)
		if result != tt.expected {
			t.Errorf("friendNames(%v) = %q, want %q", tt.friends, result, tt.expected)
		}
	}
}

// Older versions of seal stamped each piece separately, so pieces of one
// set can differ by a minute. A difference is a warning; unlocking decides.
func TestCreatedTimeMismatchIsAWarning(t *testing.T) {
	now := time.Now()
	a := &core.Share{Version: 2, Total: 3, Threshold: 2} // no timestamp (QR)
	b := &core.Share{Version: 2, Total: 3, Threshold: 2, Created: now}
	c := &core.Share{Version: 2, Total: 3, Threshold: 2, Created: now.Add(-time.Hour)}

	shares := []*core.Share{a, b, c}
	if err := validateSharesCompatible(shares); err != nil {
		t.Errorf("creation time difference refused the set: %v", err)
	}
	if w := createdTimeWarning(shares); !strings.Contains(w, "different time") {
		t.Errorf("expected a creation time warning, got %q", w)
	}
	if w := createdTimeWarning([]*core.Share{a, b}); w != "" {
		t.Errorf("matching times warned: %q", w)
	}
}
