//go:build integration

package kurabiye

import (
	"testing"
)

func TestIntegration_ChromeCookies(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"chrome"},
	})
	if err != nil {
		t.Fatalf("GetCookies() error: %v", err)
	}

	t.Logf("Got %d cookies from Chrome", len(result.Cookies))
	for _, w := range result.Warnings {
		t.Logf("Warning: %s", w)
	}
	for _, c := range result.Cookies {
		t.Logf("  %s=%s (domain=%s, path=%s, secure=%v, httpOnly=%v)",
			c.Name, c.Value, c.Domain, c.Path, c.Secure, c.HTTPOnly)
	}
}

func TestIntegration_FirefoxCookies(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"firefox"},
	})
	if err != nil {
		t.Fatalf("GetCookies() error: %v", err)
	}

	t.Logf("Got %d cookies from Firefox", len(result.Cookies))
	for _, w := range result.Warnings {
		t.Logf("Warning: %s", w)
	}
}

func TestIntegration_AllBrowsersMerge(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:  "https://google.com",
		Mode: "merge",
	})
	if err != nil {
		t.Fatalf("GetCookies() error: %v", err)
	}

	t.Logf("Got %d cookies in merge mode", len(result.Cookies))
	for _, w := range result.Warnings {
		t.Logf("Warning: %s", w)
	}

	header := ToCookieHeader(result.Cookies, true)
	t.Logf("Cookie header: %s", header)
}

func TestIntegration_FirstMode(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:  "https://google.com",
		Mode: "first",
	})
	if err != nil {
		t.Fatalf("GetCookies() error: %v", err)
	}

	t.Logf("Got %d cookies in first mode", len(result.Cookies))
	if len(result.Cookies) > 0 {
		t.Logf("Source: %s", result.Cookies[0].Source)
	}
}

func TestIntegration_NameFilter(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"chrome"},
		Names:    []string{"NID", "SID", "HSID"},
	})
	if err != nil {
		t.Fatalf("GetCookies() error: %v", err)
	}

	t.Logf("Got %d cookies matching names", len(result.Cookies))
	for _, c := range result.Cookies {
		t.Logf("  %s=%s", c.Name, c.Value)
	}
}
