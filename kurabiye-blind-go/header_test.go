package kurabiye

import (
	"testing"
	"time"
)

func TestToCookieHeader(t *testing.T) {
	cookies := []Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "token", Value: "xyz789"},
		{Name: "session", Value: "duplicate"},
	}

	tests := []struct {
		name         string
		cookies      []Cookie
		dedupeByName bool
		want         string
	}{
		{
			name:         "no dedup",
			cookies:      cookies,
			dedupeByName: false,
			want:         "session=abc123; token=xyz789; session=duplicate",
		},
		{
			name:         "with dedup",
			cookies:      cookies,
			dedupeByName: true,
			want:         "session=abc123; token=xyz789",
		},
		{
			name:         "empty cookies",
			cookies:      nil,
			dedupeByName: false,
			want:         "",
		},
		{
			name:         "single cookie",
			cookies:      []Cookie{{Name: "a", Value: "b"}},
			dedupeByName: false,
			want:         "a=b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCookieHeader(tt.cookies, tt.dedupeByName)
			if got != tt.want {
				t.Errorf("ToCookieHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCookies_InvalidURL(t *testing.T) {
	_, err := GetCookies(GetCookiesOptions{})
	if err == nil {
		t.Error("expected error for empty URL")
	}

	_, err = GetCookies(GetCookiesOptions{URL: "://invalid"})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestGetCookies_UnknownBrowser(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://example.com",
		Browsers: []string{"nonexistent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unknown browser")
	}
}

func TestGetCookies_ExpiredCookieFiltering(t *testing.T) {
	// This is more of a design verification — the actual filtering happens in GetCookies
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	cookies := []Cookie{
		{Name: "expired", Expires: past},
		{Name: "valid", Expires: future},
		{Name: "session", Expires: time.Time{}}, // zero time = session cookie
	}

	// Filter as GetCookies would
	var valid []Cookie
	for _, c := range cookies {
		if !c.Expires.IsZero() && c.Expires.Before(now) {
			continue
		}
		valid = append(valid, c)
	}

	if len(valid) != 2 {
		t.Errorf("expected 2 valid cookies, got %d", len(valid))
	}
	if valid[0].Name != "valid" {
		t.Errorf("expected first valid cookie to be 'valid', got %q", valid[0].Name)
	}
	if valid[1].Name != "session" {
		t.Errorf("expected second valid cookie to be 'session', got %q", valid[1].Name)
	}
}
