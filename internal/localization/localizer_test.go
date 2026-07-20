package localization

import "testing"

func TestTextNormalizesLocaleAndFallsBack(t *testing.T) {
	localizer, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := localizer.Text("de-DE", "about_button"); got != "Über diesen Bot" {
		t.Errorf("German text = %q", got)
	}
	if got := localizer.Text("fr", "about_button"); got != "About this bot" {
		t.Errorf("fallback text = %q", got)
	}
	if got := localizer.Text("en", "missing_key"); got != "missing_key" {
		t.Errorf("missing text = %q", got)
	}
}
