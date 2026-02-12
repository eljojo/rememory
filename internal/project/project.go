package project

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ProjectFileName = "project.yml"
	ManifestDir     = "manifest"
	OutputDir       = "output"
	SharesDir       = "shares"
)

// Friend represents a person who will hold a share.
type Friend struct {
	Name     string `yaml:"name"`
	Contact  string `yaml:"contact,omitempty"`
	Language string `yaml:"language,omitempty"` // Bundle language override (e.g. "en", "es", "de", "fr", "sl", "pt", "zh-TW")
}

// ShareInfo stores information about a generated share.
type ShareInfo struct {
	Friend   string `yaml:"friend"`
	File     string `yaml:"file"`
	Checksum string `yaml:"checksum"`
}

// SealedInfo stores information about the sealed manifest.
type Sealed struct {
	At               time.Time   `yaml:"at"`
	ManifestChecksum string      `yaml:"manifest_checksum"`
	VerificationHash string      `yaml:"verification_hash"`
	Shares           []ShareInfo `yaml:"shares"`
}

// Project represents a rememory project configuration.
type Project struct {
	Name      string   `yaml:"name"`
	Created   string   `yaml:"created"`
	Threshold int      `yaml:"threshold"`
	Anonymous bool     `yaml:"anonymous,omitempty"`
	Language  string   `yaml:"language,omitempty"` // Default bundle language (e.g. "en", "es", "de", "fr", "sl", "pt", "zh-TW")
	Friends   []Friend `yaml:"friends"`
	Sealed    *Sealed  `yaml:"sealed,omitempty"`

	// Path is the directory containing this project (not serialized)
	Path string `yaml:"-"`
}

// Load reads a project from a directory.
func Load(dir string) (*Project, error) {
	path := filepath.Join(dir, ProjectFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading project file: %w", err)
	}

	var p Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing project file: %w", err)
	}

	p.Path = dir
	return &p, nil
}

// Save writes the project configuration to disk.
func (p *Project) Save() error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("encoding project: %w", err)
	}

	path := filepath.Join(p.Path, ProjectFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing project file: %w", err)
	}

	return nil
}

// Validate checks that the project configuration is valid.
func (p *Project) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(p.Friends) < 2 {
		return fmt.Errorf("need at least 2 friends, got %d", len(p.Friends))
	}
	if p.Threshold < 2 {
		return fmt.Errorf("threshold must be at least 2, got %d", p.Threshold)
	}
	if p.Threshold > len(p.Friends) {
		return fmt.Errorf("threshold (%d) cannot exceed number of friends (%d)", p.Threshold, len(p.Friends))
	}

	for i, f := range p.Friends {
		if f.Name == "" {
			return fmt.Errorf("friend %d: name is required", i+1)
		}
	}

	return nil
}

// ManifestPath returns the path to the manifest directory.
func (p *Project) ManifestPath() string {
	return filepath.Join(p.Path, ManifestDir)
}

// OutputPath returns the path to the output directory.
func (p *Project) OutputPath() string {
	return filepath.Join(p.Path, OutputDir)
}

// SharesPath returns the path to the shares directory.
func (p *Project) SharesPath() string {
	return filepath.Join(p.Path, OutputDir, SharesDir)
}

// ManifestAgePath returns the path to the encrypted manifest.
func (p *Project) ManifestAgePath() string {
	return filepath.Join(p.Path, OutputDir, "MANIFEST.age")
}

// FindProjectDir searches up the directory tree for a project.yml file.
// Returns the directory containing the project, or an error if not found.
func FindProjectDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		projectPath := filepath.Join(dir, ProjectFileName)
		if _, err := os.Stat(projectPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", fmt.Errorf("no %s found in %s or any parent directory", ProjectFileName, startDir)
		}
		dir = parent
	}
}

// New creates a new project with the given configuration.
func New(dir, name string, threshold int, friends []Friend) (*Project, error) {
	return NewWithOptions(dir, name, threshold, friends, false)
}

// NewAnonymous creates a new anonymous project (no contact info).
func NewAnonymous(dir, name string, threshold int, numShares int) (*Project, error) {
	friends := make([]Friend, numShares)
	for i := 0; i < numShares; i++ {
		friends[i] = Friend{Name: fmt.Sprintf("Share %d", i+1)}
	}
	return NewWithOptions(dir, name, threshold, friends, true)
}

// NewWithOptions creates a new project with the given configuration.
func NewWithOptions(dir, name string, threshold int, friends []Friend, anonymous bool) (*Project, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating project directory: %w", err)
	}

	manifestDir := filepath.Join(dir, ManifestDir)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return nil, fmt.Errorf("creating manifest directory: %w", err)
	}

	p := &Project{
		Name:      name,
		Created:   time.Now().Format("2006-01-02"),
		Threshold: threshold,
		Anonymous: anonymous,
		Friends:   friends,
		Path:      dir,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := p.Save(); err != nil {
		return nil, err
	}

	return p, nil
}
