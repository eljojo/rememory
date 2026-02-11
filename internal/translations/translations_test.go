package translations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestAllJSONFilesParseCorrectly(t *testing.T) {
	for _, component := range []string{"recover", "maker", "readme"} {
		for _, lang := range Languages {
			t.Run(fmt.Sprintf("%s/%s", component, lang), func(t *testing.T) {
				m, err := GetComponentTranslations(component, lang)
				if err != nil {
					t.Fatalf("failed to load %s/%s: %v", component, lang, err)
				}
				if len(m) == 0 {
					t.Errorf("%s/%s has no translation keys", component, lang)
				}
			})
		}
	}
}

func TestAllLanguagesHaveSameKeys(t *testing.T) {
	for _, component := range []string{"recover", "maker", "readme"} {
		t.Run(component, func(t *testing.T) {
			enKeys, err := GetComponentKeys(component)
			if err != nil {
				t.Fatalf("failed to load English keys for %s: %v", component, err)
			}

			for _, lang := range Languages {
				if lang == "en" {
					continue
				}
				t.Run(lang, func(t *testing.T) {
					m, err := GetComponentTranslations(component, lang)
					if err != nil {
						t.Fatalf("failed to load %s/%s: %v", component, lang, err)
					}

					langKeys := make([]string, 0, len(m))
					for k := range m {
						langKeys = append(langKeys, k)
					}
					sort.Strings(langKeys)

					for _, key := range enKeys {
						if _, ok := m[key]; !ok {
							t.Errorf("%s/%s: missing key %q", component, lang, key)
						}
					}

					enMap := make(map[string]bool)
					for _, k := range enKeys {
						enMap[k] = true
					}
					for _, key := range langKeys {
						if !enMap[key] {
							t.Errorf("%s/%s: extra key %q not present in English", component, lang, key)
						}
					}
				})
			}
		})
	}
}

func TestGetTranslationsJSProducesValidJS(t *testing.T) {
	for _, component := range []string{"recover", "maker"} {
		t.Run(component, func(t *testing.T) {
			js := GetTranslationsJS(component)

			trimmed := strings.TrimSpace(js)
			if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
				t.Errorf("GetTranslationsJS(%s) should be wrapped in braces, got: %s...%s",
					component, trimmed[:20], trimmed[len(trimmed)-20:])
			}

			for _, lang := range Languages {
				if !strings.Contains(js, lang+": {") {
					t.Errorf("GetTranslationsJS(%s) missing language %s", component, lang)
				}
			}

			if err := json.Unmarshal([]byte(trimmed), &map[string]map[string]string{}); err != nil {
				t.Logf("Note: GetTranslationsJS output uses JS object syntax (not strict JSON), which is expected")
			}
		})
	}
}

func TestGetTranslationsJSUnknownComponent(t *testing.T) {
	js := GetTranslationsJS("nonexistent")
	if js != "{}" {
		t.Errorf("expected {} for unknown component, got: %s", js)
	}
}

func TestTWithParameterSubstitution(t *testing.T) {
	tests := []struct {
		component string
		lang      string
		key       string
		args      []any
		want      string
	}{
		{"recover", "en", "need_more", []any{3}, "Waiting for 3 more pieces"},
		{"recover", "en", "shares_of", []any{2, 5}, "2 of 5 pieces"},
		{"recover", "es", "need_more", []any{3}, "Faltan 3 partes"},
		{"recover", "en", "loading", nil, "Preparing the recovery tool..."},
		{"maker", "en", "loading", nil, "Preparing the bundle creator..."},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s", tt.component, tt.lang, tt.key), func(t *testing.T) {
			got := T(tt.component, tt.lang, tt.key, tt.args...)
			if got != tt.want {
				t.Errorf("T(%q, %q, %q, %v) = %q, want %q", tt.component, tt.lang, tt.key, tt.args, got, tt.want)
			}
		})
	}
}

func TestTFallsBackToEnglish(t *testing.T) {
	got := T("recover", "xx", "loading")
	want := "Preparing the recovery tool..."
	if got != want {
		t.Errorf("T with unknown language should fall back to English: got %q, want %q", got, want)
	}
}

func TestTFallsBackToKey(t *testing.T) {
	got := T("recover", "en", "nonexistent_key_12345")
	if got != "nonexistent_key_12345" {
		t.Errorf("T with unknown key should return key itself: got %q", got)
	}
}

func TestGetStringBasic(t *testing.T) {
	got := GetString("recover", "en", "loading")
	if got != "Preparing the recovery tool..." {
		t.Errorf("GetString(recover, en, loading) = %q, want %q", got, "Preparing the recovery tool...")
	}
}

func TestGetStringSameKeyDifferentComponents(t *testing.T) {
	recover := GetString("recover", "en", "loading")
	maker := GetString("maker", "en", "loading")
	if recover == maker {
		t.Errorf("same key 'loading' should have different values per component, both got %q", recover)
	}
	if recover != "Preparing the recovery tool..." {
		t.Errorf("recover loading = %q, want %q", recover, "Preparing the recovery tool...")
	}
	if maker != "Preparing the bundle creator..." {
		t.Errorf("maker loading = %q, want %q", maker, "Preparing the bundle creator...")
	}
}

func TestRecoverHasExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"loading", "title", "subtitle",
		"step1_title", "step2_title", "step3_title",
		"decrypt_btn", "download_btn",
		"need_more", "ready", "shares_of",
		"error_decrypt_title", "error_decrypt_message",
		"action_reload", "action_try_again",
	}

	keys, err := GetComponentKeys("recover")
	if err != nil {
		t.Fatalf("failed to get recover keys: %v", err)
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("recover is missing expected key %q", expected)
		}
	}
}

func TestMakerHasExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"loading", "title", "page_title",
		"step1_title", "step2_title", "step3_title",
		"generate_btn", "download_all_btn",
		"add_friend", "threshold_label",
		"error_title", "action_try_again",
		"language_label",
	}

	keys, err := GetComponentKeys("maker")
	if err != nil {
		t.Fatalf("failed to get maker keys: %v", err)
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("maker is missing expected key %q", expected)
		}
	}
}

func TestReadmeHasExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"title", "for", "warning_cannot_alone",
		"warning_need_friends", "warning_need_shares",
		"warning_confidential", "warning_keep_safe",
		"what_is_this", "what_bundle_for", "what_one_of", "what_threshold",
		"other_holders", "contact_label",
		"recover_browser", "recover_step1", "recover_share_loaded",
		"recover_step2", "recover_step2_drag", "recover_step2_click",
		"recover_offline", "recover_cli", "recover_cli_hint", "recover_cli_usage",
		"your_share", "recovery_words_title", "recovery_words_hint",
		"machine_readable", "metadata_footer",
		"readme_filename",
	}

	keys, err := GetComponentKeys("readme")
	if err != nil {
		t.Fatalf("failed to get readme keys: %v", err)
	}

	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("readme is missing expected key %q", expected)
		}
	}
}

func TestReadmeTranslation(t *testing.T) {
	// Test English
	got := T("readme", "en", "title")
	if got != "REMEMORY RECOVERY BUNDLE" {
		t.Errorf("readme/en/title = %q, want %q", got, "REMEMORY RECOVERY BUNDLE")
	}

	// Test Spanish
	got = T("readme", "es", "title")
	if got != "KIT DE RECUPERACIÓN REMEMORY" {
		t.Errorf("readme/es/title = %q, want %q", got, "KIT DE RECUPERACIÓN REMEMORY")
	}

	// Test parameter substitution
	got = T("readme", "en", "for", "Alice")
	if got != "For: Alice" {
		t.Errorf("readme/en/for(Alice) = %q, want %q", got, "For: Alice")
	}

	// Test fallback to English for unknown language
	got = T("readme", "xx", "title")
	if got != "REMEMORY RECOVERY BUNDLE" {
		t.Errorf("readme/xx/title should fall back to English, got %q", got)
	}
}
