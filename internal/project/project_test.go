package project

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndLoad(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	friends := []Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
	}

	// Create new project
	p, err := New(projectDir, "test-project", 2, friends)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if p.Name != "test-project" {
		t.Errorf("name: got %q, want %q", p.Name, "test-project")
	}
	if p.Threshold != 2 {
		t.Errorf("threshold: got %d, want 2", p.Threshold)
	}
	if len(p.Friends) != 2 {
		t.Errorf("friends: got %d, want 2", len(p.Friends))
	}

	// Check manifest directory was created
	manifestPath := filepath.Join(projectDir, ManifestDir)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest directory not created")
	}

	// Load the project
	loaded, err := Load(projectDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != p.Name {
		t.Errorf("loaded name: got %q, want %q", loaded.Name, p.Name)
	}
	if loaded.Threshold != p.Threshold {
		t.Errorf("loaded threshold: got %d, want %d", loaded.Threshold, p.Threshold)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		wantErr bool
	}{
		{
			name: "valid",
			project: Project{
				Name:      "test",
				Threshold: 2,
				Friends: []Friend{
					{Name: "Alice", Contact: "a@example.com"},
					{Name: "Bob", Contact: "b@example.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid without contact",
			project: Project{
				Name:      "test",
				Threshold: 2,
				Friends: []Friend{
					{Name: "Alice"},
					{Name: "Bob"},
				},
			},
			wantErr: false,
		},
		{
			name:    "no name",
			project: Project{Threshold: 2, Friends: []Friend{{Name: "A"}, {Name: "B"}}},
			wantErr: true,
		},
		{
			name:    "not enough friends",
			project: Project{Name: "test", Threshold: 2, Friends: []Friend{{Name: "A"}}},
			wantErr: true,
		},
		{
			name:    "threshold too low",
			project: Project{Name: "test", Threshold: 1, Friends: []Friend{{Name: "A"}, {Name: "B"}}},
			wantErr: true,
		},
		{
			name:    "threshold too high",
			project: Project{Name: "test", Threshold: 5, Friends: []Friend{{Name: "A"}, {Name: "B"}}},
			wantErr: true,
		},
		{
			name:    "friend missing name",
			project: Project{Name: "test", Threshold: 2, Friends: []Friend{{Contact: "a@x.com"}, {Name: "B"}}},
			wantErr: true,
		},
		{
			name: "anonymous valid without email",
			project: Project{
				Name:      "test",
				Threshold: 2,
				Anonymous: true,
				Friends: []Friend{
					{Name: "Share 1"},
					{Name: "Share 2"},
				},
			},
			wantErr: false,
		},
		{
			name: "anonymous still requires name",
			project: Project{
				Name:      "test",
				Threshold: 2,
				Anonymous: true,
				Friends: []Friend{
					{Name: ""},
					{Name: "Share 2"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindProjectDir(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure
	projectDir := filepath.Join(dir, "project")
	nestedDir := filepath.Join(projectDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create project.yml
	projectFile := filepath.Join(projectDir, ProjectFileName)
	if err := os.WriteFile(projectFile, []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Find from nested directory
	found, err := FindProjectDir(nestedDir)
	if err != nil {
		t.Fatalf("FindProjectDir: %v", err)
	}
	if found != projectDir {
		t.Errorf("got %q, want %q", found, projectDir)
	}

	// Not found
	_, err = FindProjectDir(dir)
	if err == nil {
		t.Error("expected error when project not found")
	}
}

func TestPaths(t *testing.T) {
	p := &Project{Path: "/test/project"}

	if p.ManifestPath() != "/test/project/manifest" {
		t.Errorf("ManifestPath: got %s", p.ManifestPath())
	}
	if p.OutputPath() != "/test/project/output" {
		t.Errorf("OutputPath: got %s", p.OutputPath())
	}
	if p.SharesPath() != "/test/project/output/shares" {
		t.Errorf("SharesPath: got %s", p.SharesPath())
	}
	if p.ManifestAgePath() != "/test/project/output/MANIFEST.age" {
		t.Errorf("ManifestAgePath: got %s", p.ManifestAgePath())
	}
}

func TestWriteManifestReadme(t *testing.T) {
	dir := t.TempDir()

	data := TemplateData{
		ProjectName: "test-project",
		Friends: []Friend{
			{Name: "Alice", Contact: "alice@example.com"},
			{Name: "Bob", Contact: "bob@example.com"},
		},
		Threshold: 2,
	}

	err := WriteManifestReadme(dir, data)
	if err != nil {
		t.Fatalf("WriteManifestReadme: %v", err)
	}

	// Check file exists
	readmePath := filepath.Join(dir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading README: %v", err)
	}

	// Check content contains expected values
	if !contains(string(content), "test-project") {
		t.Error("README should contain project name")
	}
	if !contains(string(content), "Alice") {
		t.Error("README should contain friend name")
	}
	if !contains(string(content), "2 of 2") {
		t.Error("README should contain threshold info")
	}
}

func TestFriendNames(t *testing.T) {
	friends := []Friend{
		{Name: "Alice"},
		{Name: "Bob"},
		{Name: "Carol"},
	}

	result := FriendNames(friends)
	if result != "Alice, Bob, Carol" {
		t.Errorf("got %q, want %q", result, "Alice, Bob, Carol")
	}

	// Empty list
	empty := FriendNames([]Friend{})
	if empty != "" {
		t.Errorf("empty friends should return empty string, got %q", empty)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent project")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	// Write invalid YAML
	if err := os.WriteFile(filepath.Join(dir, ProjectFileName), []byte("invalid: yaml: ::"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestNewInvalidProject(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "invalid")

	// Invalid: only 1 friend
	friends := []Friend{{Name: "Alice", Contact: "a@x.com"}}
	_, err := New(projectDir, "test", 2, friends)
	if err == nil {
		t.Error("expected error for invalid project")
	}
}

func TestNewAnonymous(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "anonymous-project")

	p, err := NewAnonymous(projectDir, "test-anon", 3, 5)
	if err != nil {
		t.Fatalf("NewAnonymous: %v", err)
	}

	if !p.Anonymous {
		t.Error("project should be anonymous")
	}
	if p.Name != "test-anon" {
		t.Errorf("name: got %q, want %q", p.Name, "test-anon")
	}
	if p.Threshold != 3 {
		t.Errorf("threshold: got %d, want 3", p.Threshold)
	}
	if len(p.Friends) != 5 {
		t.Errorf("friends: got %d, want 5", len(p.Friends))
	}

	// Check synthetic names
	for i, f := range p.Friends {
		expected := fmt.Sprintf("Share %d", i+1)
		if f.Name != expected {
			t.Errorf("friend %d name: got %q, want %q", i, f.Name, expected)
		}
		if f.Contact != "" {
			t.Errorf("friend %d should have no contact", i)
		}
	}
}

func TestNewWithOptions(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "options-project")

	friends := []Friend{
		{Name: "Share 1"},
		{Name: "Share 2"},
		{Name: "Share 3"},
	}

	p, err := NewWithOptions(projectDir, "test-options", 2, friends, true)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	if !p.Anonymous {
		t.Error("project should be anonymous")
	}
	if len(p.Friends) != 3 {
		t.Errorf("friends: got %d, want 3", len(p.Friends))
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test")

	friends := []Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
	}

	p, err := New(projectDir, "test", 2, friends)
	if err != nil {
		t.Fatal(err)
	}

	// Modify and save
	p.Sealed = &Sealed{
		ManifestChecksum: "sha256:abc",
		VerificationHash: "sha256:def",
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload
	loaded, err := Load(projectDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Sealed == nil {
		t.Fatal("Sealed should not be nil")
	}
	if loaded.Sealed.ManifestChecksum != "sha256:abc" {
		t.Errorf("ManifestChecksum: got %q", loaded.Sealed.ManifestChecksum)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
