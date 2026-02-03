package cmd

import (
	"testing"

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
