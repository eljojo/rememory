package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/eljojo/rememory/internal/project"
)

func TestPrintMessages_Initial(t *testing.T) {
	p := &project.Project{
		Name:      "test-safe",
		Threshold: 2,
		Friends: []project.Friend{
			{Name: "Alice", Contact: "alice@email.com"},
			{Name: "Bob", Contact: "555-1234"},
			{Name: "Camila"},
		},
		Sealed: &project.Sealed{},
		Path:   t.TempDir(),
	}

	// Create output/bundles directory (printMessages checks for bundle files)
	os.MkdirAll(p.OutputPath()+"/bundles", 0755)

	output := captureStdout(t, func() {
		err := printMessages(p, false, "")
		if err != nil {
			t.Fatalf("printMessages failed: %v", err)
		}
	})

	// Check all friends appear
	if !strings.Contains(output, "Alice") {
		t.Error("output missing Alice")
	}
	if !strings.Contains(output, "Bob") {
		t.Error("output missing Bob")
	}
	if !strings.Contains(output, "Camila") {
		t.Error("output missing Camila")
	}

	// Check bundle filenames
	if !strings.Contains(output, "bundle-alice.zip") {
		t.Error("output missing bundle-alice.zip")
	}

	// Check contact info appears in header
	if !strings.Contains(output, "alice@email.com") {
		t.Error("output missing alice@email.com")
	}
	if !strings.Contains(output, "555-1234") {
		t.Error("output missing 555-1234")
	}

	// Check threshold and total are mentioned
	if !strings.Contains(output, "3 people") {
		t.Error("output missing total count (3 people)")
	}
	if !strings.Contains(output, "any 2 of you") {
		t.Error("output missing threshold (any 2)")
	}

	// Check it doesn't mention "rememory" in the message body
	// (the tool name should stay out of friend-facing messages)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "rememory") && !strings.Contains(lower, "bundle:") {
			// Allow "rememory" in the header/bundle filename, not in the message body
			if !strings.HasPrefix(strings.TrimSpace(line), "Bundle:") &&
				!strings.Contains(line, "\u2500") {
				t.Errorf("message body mentions 'rememory': %s", line)
			}
		}
	}
}

func TestPrintMessages_Rotation(t *testing.T) {
	p := &project.Project{
		Name:      "test-safe",
		Threshold: 2,
		Friends: []project.Friend{
			{Name: "Alice"},
			{Name: "Bob"},
		},
		Sealed: &project.Sealed{},
		Path:   t.TempDir(),
	}
	os.MkdirAll(p.OutputPath()+"/bundles", 0755)

	output := captureStdout(t, func() {
		err := printMessages(p, true, "")
		if err != nil {
			t.Fatalf("printMessages failed: %v", err)
		}
	})

	// Rotation messages should be shorter
	if !strings.Contains(output, "updated") {
		t.Error("rotation message missing 'updated'")
	}
	if !strings.Contains(output, "no longer works") {
		t.Error("rotation message missing 'no longer works'")
	}

	// Should NOT contain the full initial explanation
	if strings.Contains(output, "recover.html") {
		t.Error("rotation message should not re-explain recover.html")
	}
}

func TestPrintMessages_FriendFilter(t *testing.T) {
	p := &project.Project{
		Name:      "test-safe",
		Threshold: 2,
		Friends: []project.Friend{
			{Name: "Alice"},
			{Name: "Bob"},
			{Name: "Camila"},
		},
		Sealed: &project.Sealed{},
		Path:   t.TempDir(),
	}
	os.MkdirAll(p.OutputPath()+"/bundles", 0755)

	output := captureStdout(t, func() {
		err := printMessages(p, false, "Bob")
		if err != nil {
			t.Fatalf("printMessages failed: %v", err)
		}
	})

	if !strings.Contains(output, "Bob") {
		t.Error("filtered output missing Bob")
	}
	if strings.Contains(output, "Hi Alice") {
		t.Error("filtered output should not contain Alice's message")
	}
	if strings.Contains(output, "Hi Camila") {
		t.Error("filtered output should not contain Camila's message")
	}
}

func TestPrintMessages_FriendNotFound(t *testing.T) {
	p := &project.Project{
		Name:      "test-safe",
		Threshold: 2,
		Friends: []project.Friend{
			{Name: "Alice"},
		},
		Sealed: &project.Sealed{},
		Path:   t.TempDir(),
	}
	os.MkdirAll(p.OutputPath()+"/bundles", 0755)

	err := printMessages(p, false, "Nobody")
	if err == nil {
		t.Error("expected error for unknown friend")
	}
	if !strings.Contains(err.Error(), "Nobody") {
		t.Errorf("error should mention the friend name: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
