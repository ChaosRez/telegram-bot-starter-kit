// Package localization provides embedded, dependency-light translations.
package localization

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultLocale = "en"

//go:embed locales/*.yaml
var localeFiles embed.FS

// Localizer resolves messages using a locale with English fallback.
type Localizer struct {
	messages map[string]map[string]string
}

// New loads and validates embedded locale files.
func New() (*Localizer, error) {
	localizer := &Localizer{messages: make(map[string]map[string]string)}

	files, err := fs.Glob(localeFiles, "locales/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("list locale files: %w", err)
	}
	for _, name := range files {
		contents, err := localeFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read locale file %s: %w", name, err)
		}

		translations := make(map[string]string)
		if err := yaml.Unmarshal(contents, &translations); err != nil {
			return nil, fmt.Errorf("decode locale file %s: %w", name, err)
		}

		locale := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		localizer.messages[locale] = translations
	}

	if _, ok := localizer.messages[defaultLocale]; !ok {
		return nil, fmt.Errorf("default locale %q is missing", defaultLocale)
	}

	return localizer, nil
}

// Text returns a translated message, falling back to English and then the key.
func (localizer *Localizer) Text(locale, key string) string {
	locale = normalize(locale)
	if translations, ok := localizer.messages[locale]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}
	if value, ok := localizer.messages[defaultLocale][key]; ok {
		return value
	}
	return key
}

func normalize(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if separator := strings.IndexAny(locale, "-_"); separator >= 0 {
		locale = locale[:separator]
	}
	if locale == "" {
		return defaultLocale
	}
	return locale
}
