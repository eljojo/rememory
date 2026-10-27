package bundle

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/translations"
)

func testReadmeData() ReadmeData {
	share := core.NewShare(1, 1, 3, 2, "Alice", []byte("test-share-data-for-readme-123456"))
	return ReadmeData{
		ProjectName:      "Test Project",
		Holder:           "Alice",
		Share:            share,
		OtherFriends:     []project.Friend{{Name: "Bob", Contact: "bob@example.com"}, {Name: "Carol"}},
		Threshold:        2,
		Total:            3,
		Version:          "v0.0.1-test",
		GitHubReleaseURL: "https://github.com/eljojo/rememory/releases",
		ManifestChecksum: "sha256:abcdef1234567890",
		RecoverChecksum:  "sha256:0987654321fedcba",
		Created:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Language:         "en",
	}
}

func en(key string, args ...any) string {
	return translations.T("readme", "en", key, args...)
}

// Every friend reads the top; the two situations (being asked for a piece,
// bringing the pieces together) follow, then the piece itself. The program
// for helpers and the machine metadata come last.
func TestReadmeSectionsFollowTheFriend(t *testing.T) {
	readme := GenerateReadme(testReadmeData())

	order := []string{
		en("what_is_this"),
		en("warning_title"),
		en("other_holders"),
		en("sharing_title"),
		en("your_share"),
		en("recover_browser"),
		en("recover_cli"),
		"METADATA FOOTER",
	}
	last := -1
	for _, heading := range order {
		i := strings.Index(readme, heading)
		if i < 0 {
			t.Fatalf("README is missing %q", heading)
		}
		if i < last {
			t.Errorf("%q comes too early in the README", heading)
		}
		last = i
	}
}

// When the locked files are inside recover.html there's nothing to add, so
// the README shouldn't send anyone looking for MANIFEST.age.
func TestReadmeMentionsLockedFilesOnlyWhenSeparate(t *testing.T) {
	data := testReadmeData()

	data.ManifestEmbedded = true
	if readme := GenerateReadme(data); strings.Contains(readme, "MANIFEST.age") {
		t.Error("embedded README mentions MANIFEST.age")
	}

	data.ManifestEmbedded = false
	if readme := GenerateReadme(data); !strings.Contains(readme, "MANIFEST.age") {
		t.Error("README for a bundle with a separate MANIFEST.age doesn't mention it")
	}
}

// Steps are numbered by the generator, so skipping one (no locked files to
// add, no contact list) never leaves a gap.
func TestReadmeStepsAreNumberedWithoutGaps(t *testing.T) {
	cases := map[string]func(*ReadmeData){
		"embedded":      func(d *ReadmeData) { d.ManifestEmbedded = true },
		"separate":      func(d *ReadmeData) { d.ManifestEmbedded = false },
		"anonymous":     func(d *ReadmeData) { d.Anonymous = true; d.OtherFriends = nil; d.ManifestEmbedded = true },
		"anon-separate": func(d *ReadmeData) { d.Anonymous = true; d.OtherFriends = nil },
	}
	step := regexp.MustCompile(`(?m)^(\d+)\. `)
	for name, tweak := range cases {
		t.Run(name, func(t *testing.T) {
			data := testReadmeData()
			tweak(&data)
			readme := GenerateReadme(data)
			start := strings.Index(readme, en("recover_browser"))
			end := strings.Index(readme, en("recover_cli"))
			steps := step.FindAllStringSubmatch(readme[start:end], -1)
			if len(steps) < 3 {
				t.Fatalf("expected at least 3 steps, got %d", len(steps))
			}
			for i, s := range steps {
				if want := strconv.Itoa(i + 1); s[1] != want {
					t.Errorf("step %s is numbered %s", want, s[1])
				}
			}
		})
	}
}

// Anonymous bundles have no list of names, so no step can point at one.
func TestReadmeAnonymousHasNoContactStep(t *testing.T) {
	data := testReadmeData()
	data.Anonymous = true
	data.OtherFriends = nil
	readme := GenerateReadme(data)
	if strings.Contains(readme, en("recover_step_contact")) {
		t.Error("anonymous README points at a contact list it doesn't have")
	}
	if strings.Contains(readme, en("recover_step_checkmarks")) {
		t.Error("anonymous README mentions checkmarks next to names")
	}
}

// The command-line example must work for every bundle. Reading the locked
// files from recover.html fails when they ship as MANIFEST.age instead.
func TestReadmeProgramExampleWorksForEveryBundle(t *testing.T) {
	readme := GenerateReadme(testReadmeData())
	if strings.Contains(readme, "--manifest recover.html") {
		t.Error("README still suggests --manifest recover.html")
	}
	if !strings.Contains(readme, "rememory recover ") {
		t.Error("README has no command-line example")
	}
}

// Friends read this calmly or in a hard moment; nothing should shout.
func TestReadmeDoesNotShout(t *testing.T) {
	readme := GenerateReadme(testReadmeData())
	for _, s := range []string{"!!", "AUTOMATICALLY", "ALREADY LOADED", "PRIMARY METHOD", "FALLBACK"} {
		if strings.Contains(readme, s) {
			t.Errorf("README contains %q", s)
		}
	}
}
