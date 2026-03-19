package kurabiye

import (
	"testing"
	"time"
)

func TestToCookieHeader(t *testing.T) {
	cookies := []Cookie{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
		{Name: "a", Value: "3"},
	}

	got := ToCookieHeader(cookies, false)
	want := "a=1; b=2; a=3"
	if got != want {
		t.Errorf("ToCookieHeader(dedupe=false) = %q, want %q", got, want)
	}

	got = ToCookieHeader(cookies, true)
	want = "a=1; b=2"
	if got != want {
		t.Errorf("ToCookieHeader(dedupe=true) = %q, want %q", got, want)
	}
}

func TestToCookieHeaderEmpty(t *testing.T) {
	got := ToCookieHeader(nil, false)
	if got != "" {
		t.Errorf("ToCookieHeader(nil) = %q, want empty", got)
	}
}

func TestChromiumTimestampToTime(t *testing.T) {
	// Chrome timestamp for 2024-01-01 00:00:00 UTC
	// Unix seconds: 1704067200
	// Chrome microseconds: (1704067200 + 11644473600) * 1000000
	chromeUsec := int64((1704067200 + 11644473600) * 1000000)
	got := chromiumTimestampToTime(chromeUsec)
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Errorf("chromiumTimestampToTime(%d) = %v, want %v", chromeUsec, got, want)
	}
}

func TestChromiumTimestampZero(t *testing.T) {
	got := chromiumTimestampToTime(0)
	if !got.IsZero() {
		t.Errorf("chromiumTimestampToTime(0) should be zero time, got %v", got)
	}
}

func TestChromiumSameSite(t *testing.T) {
	tests := []struct {
		v    int
		want string
	}{
		{-1, "None"},
		{0, ""},
		{1, "Lax"},
		{2, "Strict"},
		{99, ""},
	}
	for _, tt := range tests {
		got := chromiumSameSite(tt.v)
		if got != tt.want {
			t.Errorf("chromiumSameSite(%d) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestFirefoxSameSite(t *testing.T) {
	tests := []struct {
		v    int
		want string
	}{
		{0, "None"},
		{1, "Lax"},
		{2, "Strict"},
		{99, ""},
	}
	for _, tt := range tests {
		got := firefoxSameSite(tt.v)
		if got != tt.want {
			t.Errorf("firefoxSameSite(%d) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestGetCookiesRequiresURL(t *testing.T) {
	_, err := GetCookies(GetCookiesOptions{})
	if err == nil {
		t.Error("GetCookies with empty URL should return error")
	}
}

func TestGetCookiesInvalidURL(t *testing.T) {
	_, err := GetCookies(GetCookiesOptions{URL: "not a url"})
	if err == nil {
		t.Error("GetCookies with invalid URL should return error")
	}
}
