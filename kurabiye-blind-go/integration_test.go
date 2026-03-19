//go:build integration

package kurabiye

import (
	"fmt"
	"testing"
)

func TestIntegrationChromeGetCookies(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"chrome"},
	})
	if err != nil {
		t.Fatalf("GetCookies error: %v", err)
	}

	for _, w := range result.Warnings {
		t.Logf("warning: %s", w)
	}

	t.Logf("found %d cookies from Chrome for google.com", len(result.Cookies))
	for _, c := range result.Cookies {
		t.Logf("  %s=%s (domain=%s, expires=%v)", c.Name, truncate(c.Value, 20), c.Domain, c.Expires)
	}

	if len(result.Cookies) == 0 {
		t.Log("no cookies found — this may be expected if Chrome has no google.com cookies")
	}
}

func TestIntegrationFirefoxGetCookies(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"firefox"},
	})
	if err != nil {
		t.Fatalf("GetCookies error: %v", err)
	}

	for _, w := range result.Warnings {
		t.Logf("warning: %s", w)
	}

	t.Logf("found %d cookies from Firefox for google.com", len(result.Cookies))
}

func TestIntegrationMergeMode(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:  "https://google.com",
		Mode: "merge",
	})
	if err != nil {
		t.Fatalf("GetCookies error: %v", err)
	}

	for _, w := range result.Warnings {
		t.Logf("warning: %s", w)
	}

	t.Logf("found %d cookies across all browsers", len(result.Cookies))

	header := ToCookieHeader(result.Cookies, true)
	if header != "" {
		t.Logf("Cookie header: %s", truncate(header, 100))
	}
}

func TestIntegrationCLIHeader(t *testing.T) {
	result, err := GetCookies(GetCookiesOptions{
		URL:      "https://google.com",
		Browsers: []string{"chrome"},
	})
	if err != nil {
		t.Fatalf("GetCookies error: %v", err)
	}

	header := ToCookieHeader(result.Cookies, true)
	fmt.Printf("Cookie: %s\n", header)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
