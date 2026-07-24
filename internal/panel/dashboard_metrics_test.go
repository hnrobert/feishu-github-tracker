package panel

import (
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

func TestSummarizeDeliveries(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.Local)
	lines := []string{
		"2026/07/24 09:00:00 [INFO] Event matched: push, sending notification",
		"2026/07/24 09:00:01 [INFO] Successfully sent notification to dev-team",
		"2026/07/24 10:00:00 [INFO] Event matched: issues, sending notification",
		"2026/07/24 10:00:01 [ERROR] Failed to send notification to ops-team: unavailable",
		"2026/07/17 10:00:01 [INFO] Successfully sent notification to outside-window",
	}

	got := summarizeDeliveries(lines, now)
	if got.Total != 2 || got.Failed != 1 || got.SuccessRate != 50 {
		t.Fatalf("unexpected summary: %#v", got)
	}
	if len(got.Recent) != 2 || got.Recent[0].Target != "ops-team" || got.Recent[0].Success {
		t.Fatalf("unexpected recent deliveries: %#v", got.Recent)
	}
	if len(got.Events) != 2 || got.Events[0].Count != 1 {
		t.Fatalf("unexpected events: %#v", got.Events)
	}
}

func TestLocaleFromAndTranslate(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	if got := localeFrom(r); got != "en-US" {
		t.Fatalf("localeFrom() = %q, want en-US", got)
	}
	if got := translate(ViewData{Locale: "en-US"}, "nav.overview"); got != "Overview" {
		t.Fatalf("translate() = %q", got)
	}
	if got := translate(ViewData{Locale: "invalid"}, "nav.overview"); got != "概览" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestLocaleCatalogsHaveMatchingKeys(t *testing.T) {
	for key := range messages["zh-CN"] {
		if messages["en-US"][key] == "" {
			t.Errorf("en-US is missing key %q", key)
		}
	}
	for key := range messages["en-US"] {
		if messages["zh-CN"][key] == "" {
			t.Errorf("zh-CN is missing key %q", key)
		}
	}
}

func TestTemplateLocaleKeysExistAndHaveNoMixedLanguageCopy(t *testing.T) {
	keyPattern := regexp.MustCompile(`\{\{t\s+[$.]\s+"([^"]+)"`)
	mixedCopy := regexp.MustCompile(`[\p{Han}]\s*/|/\s*[\p{Han}]`)
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := templatesFS.ReadFile("templates/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range keyPattern.FindAllStringSubmatch(string(data), -1) {
			for locale, catalog := range messages {
				if catalog[match[1]] == "" {
					t.Errorf("%s references %q, missing in %s", entry.Name(), match[1], locale)
				}
			}
		}
		if mixedCopy.Match(data) {
			t.Errorf("%s still contains mixed-language UI copy", entry.Name())
		}
	}
}
