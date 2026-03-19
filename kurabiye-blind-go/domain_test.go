package kurabiye

import "testing"

func TestParseDomain(t *testing.T) {
	tests := []struct {
		url    string
		want   string
		hasErr bool
	}{
		{"https://twitter.com/", "twitter.com", false},
		{"https://www.Twitter.COM/path?q=1", "www.twitter.com", false},
		{"http://localhost:8080/foo", "localhost", false},
		{"https://sub.domain.example.com", "sub.domain.example.com", false},
		{"not a url", "", true},
	}
	for _, tt := range tests {
		got, err := parseDomain(tt.url)
		if tt.hasErr {
			if err == nil {
				t.Errorf("parseDomain(%q) expected error, got %q", tt.url, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDomain(%q) unexpected error: %v", tt.url, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDomain(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestDomainMatches(t *testing.T) {
	tests := []struct {
		host   string
		cookie string
		want   bool
	}{
		// Exact match
		{"twitter.com", "twitter.com", true},
		{"twitter.com", "Twitter.COM", true},

		// Leading dot = suffix match
		{".twitter.com", ".twitter.com", true},
		{"twitter.com", ".twitter.com", true},
		{"www.twitter.com", ".twitter.com", true},
		{"sub.www.twitter.com", ".twitter.com", true},

		// No match
		{"evil-twitter.com", ".twitter.com", false},
		{"twitter.com.evil.com", ".twitter.com", false},
		{"example.com", "twitter.com", false},
		{"example.com", ".twitter.com", false},

		// Exact match without dot
		{"twitter.com", "twitter.com", true},
		{"www.twitter.com", "twitter.com", false},

		// IP addresses
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.1", ".0.0.1", false},
	}
	for _, tt := range tests {
		got := domainMatches(tt.host, tt.cookie)
		if got != tt.want {
			t.Errorf("domainMatches(%q, %q) = %v, want %v", tt.host, tt.cookie, got, tt.want)
		}
	}
}
