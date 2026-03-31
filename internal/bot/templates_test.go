package bot

import (
	"fmt"
	"strings"
	"testing"
)

func TestRandomTemplate(t *testing.T) {
	templates := []string{"hello %s", "hi %s", "hey %s"}
	result := randomTemplate(templates, "world")

	valid := false
	for _, tmpl := range templates {
		if result == fmt.Sprintf(tmpl, "world") {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("randomTemplate returned unexpected result: %q", result)
	}
}

func TestRandomTemplateVariety(t *testing.T) {
	templates := []string{"a %s", "b %s", "c %s", "d %s", "e %s"}
	seen := make(map[string]bool)

	for range 100 {
		result := randomTemplate(templates, "x")
		seen[result] = true
	}

	if len(seen) < 2 {
		t.Errorf("randomTemplate produced only %d unique results in 100 calls, expected variety", len(seen))
	}
}

func TestConfirmationTemplates(t *testing.T) {
	for i, tmpl := range confirmationTemplates {
		result := fmt.Sprintf(tmpl, "02/Jan/2027", "testuser")
		if !strings.Contains(result, "02/Jan/2027") {
			t.Errorf("confirmationTemplates[%d]: missing date in result: %q", i, result)
		}
		if !strings.Contains(result, "@testuser") {
			t.Errorf("confirmationTemplates[%d]: missing @handle in result: %q", i, result)
		}
		if len(result) > maxTweetLength {
			t.Errorf("confirmationTemplates[%d]: exceeds max tweet length (%d > %d): %q", i, len(result), maxTweetLength, result)
		}
	}
}

func TestQuoteTweetTemplates(t *testing.T) {
	for i, tmpl := range quoteTweetTemplates {
		result := fmt.Sprintf(tmpl, 3, "testuser")
		if !strings.Contains(result, "3") {
			t.Errorf("quoteTweetTemplates[%d]: missing years in result: %q", i, result)
		}
		if !strings.Contains(result, "@testuser") {
			t.Errorf("quoteTweetTemplates[%d]: missing @handle in result: %q", i, result)
		}
		if len(result) > maxTweetLength {
			t.Errorf("quoteTweetTemplates[%d]: exceeds max tweet length (%d > %d): %q", i, len(result), maxTweetLength, result)
		}
	}
}

func TestRepostTemplates(t *testing.T) {
	for i, tmpl := range repostTemplates {
		result := fmt.Sprintf(tmpl, "testuser", 2, "some tweet text", "1234567890")
		if !strings.Contains(result, "@testuser") {
			t.Errorf("repostTemplates[%d]: missing @handle in result: %q", i, result)
		}
		if !strings.Contains(result, "2") {
			t.Errorf("repostTemplates[%d]: missing years in result: %q", i, result)
		}
		if !strings.Contains(result, "some tweet text") {
			t.Errorf("repostTemplates[%d]: missing tweet text in result: %q", i, result)
		}
		if !strings.Contains(result, "1234567890") {
			t.Errorf("repostTemplates[%d]: missing tweet ID in result: %q", i, result)
		}
	}
}

func TestDeletedTemplates(t *testing.T) {
	for i, tmpl := range deletedTemplates {
		result := fmt.Sprintf(tmpl, "testuser", 5, "original text here", "9876543210")
		if !strings.Contains(result, "@testuser") {
			t.Errorf("deletedTemplates[%d]: missing @handle in result: %q", i, result)
		}
		if !strings.Contains(result, "5") {
			t.Errorf("deletedTemplates[%d]: missing years in result: %q", i, result)
		}
		if !strings.Contains(result, "original text here") {
			t.Errorf("deletedTemplates[%d]: missing tweet text in result: %q", i, result)
		}
		if !strings.Contains(result, "9876543210") {
			t.Errorf("deletedTemplates[%d]: missing tweet ID in result: %q", i, result)
		}
	}
}

func TestRepostTemplatePrefixLength(t *testing.T) {
	// Verify that computing prefix with empty text/ID gives a correct baseline for truncation
	for i, tmpl := range repostTemplates {
		prefix := fmt.Sprintf(tmpl, "longusername", 5, "", "")
		if len(prefix) == 0 {
			t.Errorf("repostTemplates[%d]: prefix is empty", i)
		}
	}
}

func TestDeletedTemplatePrefixLength(t *testing.T) {
	for i, tmpl := range deletedTemplates {
		prefix := fmt.Sprintf(tmpl, "longusername", 5, "", "")
		if len(prefix) == 0 {
			t.Errorf("deletedTemplates[%d]: prefix is empty", i)
		}
	}
}

func TestTemplatePoolsNotEmpty(t *testing.T) {
	pools := map[string][]string{
		"confirmation": confirmationTemplates,
		"quoteTweet":   quoteTweetTemplates,
		"repost":       repostTemplates,
		"deleted":      deletedTemplates,
	}
	for name, pool := range pools {
		if len(pool) < 2 {
			t.Errorf("%s template pool has fewer than 2 templates: %d", name, len(pool))
		}
	}
}
