package herotest

import "testing"

func TestClassifyAssertionURL(t *testing.T) {
	for _, criterion := range []string{
		"URL should contain /dashboard",
		"User is redirected to login page",
		"Navigate to the settings page",
		"Route changes to /home",
	} {
		hint := ClassifyAssertion(criterion)
		if hint.Kind != "url" {
			t.Errorf("ClassifyAssertion(%q).Kind = %q, want %q", criterion, hint.Kind, "url")
		}
		if hint.Criterion != criterion {
			t.Errorf("ClassifyAssertion(%q).Criterion = %q, want original", criterion, hint.Criterion)
		}
	}
}

func TestClassifyAssertionVisible(t *testing.T) {
	tests := []struct {
		criterion string
		selector  string
	}{
		{"Dashboard displays user metrics", ""},
		{"Show `#dashboard` element", "#dashboard"},
		{"Error message should appear", ""},
		{"Render the `.main-nav` component", ".main-nav"},
	}
	for _, tt := range tests {
		hint := ClassifyAssertion(tt.criterion)
		if hint.Kind != "visible" {
			t.Errorf("ClassifyAssertion(%q).Kind = %q, want %q", tt.criterion, hint.Kind, "visible")
		}
		if hint.Selector != tt.selector {
			t.Errorf("ClassifyAssertion(%q).Selector = %q, want %q", tt.criterion, hint.Selector, tt.selector)
		}
	}
}

func TestClassifyAssertionTextContains(t *testing.T) {
	tests := []struct {
		criterion  string
		quotedText string
	}{
		{`Page contains text "Welcome"`, "Welcome"},
		{`Label says 'Hello'`, "Hello"},
		{`The message includes certain content`, ""},
	}
	for _, tt := range tests {
		hint := ClassifyAssertion(tt.criterion)
		if hint.Kind != "text_contains" {
			t.Errorf("ClassifyAssertion(%q).Kind = %q, want %q", tt.criterion, hint.Kind, "text_contains")
		}
		if hint.QuotedText != tt.quotedText {
			t.Errorf("ClassifyAssertion(%q).QuotedText = %q, want %q", tt.criterion, hint.QuotedText, tt.quotedText)
		}
	}
}

func TestClassifyAssertionError(t *testing.T) {
	for _, criterion := range []string{
		"The request fails with a 403 error",
		"Request is rejected when unauthorized",
		"System should fail gracefully",
	} {
		hint := ClassifyAssertion(criterion)
		if hint.Kind != "error" {
			t.Errorf("ClassifyAssertion(%q).Kind = %q, want %q", criterion, hint.Kind, "error")
		}
	}
}

func TestClassifyAssertionCount(t *testing.T) {
	hint := ClassifyAssertion("Results count should match expected")
	if hint.Kind != "count" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "count")
	}
	hint = ClassifyAssertion("Number of items is 5")
	if hint.Kind != "count" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "count")
	}
}

func TestClassifyAssertionInput(t *testing.T) {
	hint := ClassifyAssertion("Form input field is pre-filled")
	if hint.Kind != "input" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "input")
	}
}

func TestClassifyAssertionClick(t *testing.T) {
	hint := ClassifyAssertion("Click the submit button")
	if hint.Kind != "click" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "click")
	}
}

func TestClassifyAssertionTitle(t *testing.T) {
	hint := ClassifyAssertion("Page title should be Dashboard")
	if hint.Kind != "title" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "title")
	}
}

func TestClassifyAssertionOutput(t *testing.T) {
	hint := ClassifyAssertion("API returns JSON with user data")
	if hint.Kind != "output" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "output")
	}
}

func TestClassifyAssertionDefault(t *testing.T) {
	hint := ClassifyAssertion("Something completely unmappable happens")
	if hint.Kind != "default" {
		t.Errorf("Kind = %q, want %q", hint.Kind, "default")
	}
}
