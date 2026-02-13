package translations

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed recover/*.json
var recoverFS embed.FS

//go:embed maker/*.json
var makerFS embed.FS

//go:embed readme/*.json
var readmeFS embed.FS

// Languages lists all supported language codes.
var Languages = []string{"en", "es", "de", "fr", "sl", "pt", "zh-TW"}

// LangNames maps language codes to their native display names, in the same
// order as Languages. This is the single source of truth for the language
// picker UI — HTML templates use the generator functions below instead of
// hardcoding option lists.
var LangNames = [][2]string{
	{"en", "English"},
	{"es", "Español"},
	{"de", "Deutsch"},
	{"fr", "Français"},
	{"sl", "Slovenščina"},
	{"pt", "Português"},
	{"zh-TW", "中文（台灣）"},
}

// LangSelectOptions returns HTML <option> elements for all languages,
// suitable for injection into a <select> element.
func LangSelectOptions() string {
	var b strings.Builder
	for i, entry := range LangNames {
		if i > 0 {
			b.WriteString("\n        ")
		}
		b.WriteString(`<option value="`)
		b.WriteString(entry[0])
		b.WriteString(`">`)
		b.WriteString(entry[1])
		b.WriteString(`</option>`)
	}
	return b.String()
}

// LangDetectJS returns a JavaScript array literal of non-English language codes
// for use in navigator.languages detection, e.g. ['es','de','fr','sl','pt','zh-TW'].
func LangDetectJS() string {
	var codes []string
	for _, entry := range LangNames {
		if entry[0] != "en" {
			codes = append(codes, "'"+entry[0]+"'")
		}
	}
	return "[" + strings.Join(codes, ",") + "]"
}

// GetTranslationsJS builds the JavaScript translations object for injection into HTML templates.
// component must be "recover", "maker", or "readme".
// Returns a string like: { en: {...}, es: {...}, de: {...}, fr: {...}, sl: {...} }
func GetTranslationsJS(component string) string {
	fs := fsForComponent(component)
	if fs == nil {
		return "{}"
	}

	var parts []string
	for _, lang := range Languages {
		data, err := fs.ReadFile(component + "/" + lang + ".json")
		if err != nil {
			continue
		}
		// Validate it's valid JSON, then output it as-is (preserving formatting)
		var check map[string]string
		if err := json.Unmarshal(data, &check); err != nil {
			continue
		}
		// Re-marshal to produce compact JS-friendly output with sorted keys
		keys := make([]string, 0, len(check))
		for k := range check {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var entries []string
		for _, k := range keys {
			keyJSON, _ := json.Marshal(k)
			valJSON, _ := json.Marshal(check[k])
			entries = append(entries, fmt.Sprintf("        %s: %s", string(keyJSON), string(valJSON)))
		}
		langJSON, _ := json.Marshal(lang)
		parts = append(parts, fmt.Sprintf("      %s: {\n%s\n      }", langJSON, strings.Join(entries, ",\n")))
	}

	return "{\n" + strings.Join(parts, ",\n\n") + "\n    }"
}

// T returns the translated string for the given component, language, and key,
// with {0}, {1}, ... parameter substitution.
// Falls back to English if the key is not found in the requested language.
// Falls back to the key itself if not found in any language.
func T(component, lang, key string, args ...any) string {
	text := GetString(component, lang, key)
	for i, arg := range args {
		text = strings.Replace(text, fmt.Sprintf("{%d}", i), fmt.Sprint(arg), 1)
	}
	return text
}

// GetString returns the raw translated string for the given component, language, and key.
// Falls back to English, then to the key itself.
func GetString(component, lang, key string) string {
	cache := loadComponentCache(component)
	if langMap, ok := cache[lang]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}
	if langMap, ok := cache["en"]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}
	return key
}

// componentCache caches loaded translations per component.
// Key is component name, value maps lang -> key -> value.
var componentCache map[string]map[string]map[string]string

func loadComponentCache(component string) map[string]map[string]string {
	if componentCache == nil {
		componentCache = make(map[string]map[string]map[string]string)
	}
	if cache, ok := componentCache[component]; ok {
		return cache
	}
	cache := make(map[string]map[string]string)
	fs := fsForComponent(component)
	if fs != nil {
		for _, lang := range Languages {
			data, err := fs.ReadFile(component + "/" + lang + ".json")
			if err != nil {
				continue
			}
			var m map[string]string
			if err := json.Unmarshal(data, &m); err != nil {
				continue
			}
			cache[lang] = m
		}
	}
	componentCache[component] = cache
	return cache
}

// GetComponentTranslations returns the translations map for a specific component and language.
func GetComponentTranslations(component, lang string) (map[string]string, error) {
	fs := fsForComponent(component)
	if fs == nil {
		return nil, fmt.Errorf("unknown component: %s", component)
	}
	data, err := fs.ReadFile(component + "/" + lang + ".json")
	if err != nil {
		return nil, fmt.Errorf("language %s not found for component %s", lang, component)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid JSON for %s/%s: %w", component, lang, err)
	}
	return m, nil
}

// GetComponentKeys returns all translation keys for a component (using English as reference).
func GetComponentKeys(component string) ([]string, error) {
	m, err := GetComponentTranslations(component, "en")
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// ReadmeFilename returns the translated README filename for a given language and extension.
// e.g. ReadmeFilename("es", ".txt") returns "LEEME.txt"
func ReadmeFilename(lang, ext string) string {
	name := GetString("readme", lang, "readme_filename")
	return name + ext
}

// IsReadmeFile checks whether a filename matches any translated README filename
// with the given extension (e.g. ".txt" or ".pdf").
func IsReadmeFile(filename, ext string) bool {
	for _, lang := range Languages {
		if filename == ReadmeFilename(lang, ext) {
			return true
		}
	}
	return false
}

func fsForComponent(component string) *embed.FS {
	switch component {
	case "recover":
		return &recoverFS
	case "maker":
		return &makerFS
	case "readme":
		return &readmeFS
	default:
		return nil
	}
}
