package kurabiye

import "testing"

func TestDomainMatches(t *testing.T) {
	tests := []struct {
		name          string
		cookieDomain  string
		requestDomain string
		want          bool
	}{
		// Exact matches
		{"exact match", "example.com", "example.com", true},
		{"exact match with dot prefix", ".example.com", "example.com", true},

		// Subdomain matches
		{"subdomain match", ".example.com", "www.example.com", true},
		{"subdomain match no dot", "example.com", "www.example.com", true},
		{"deep subdomain match", ".example.com", "a.b.example.com", true},

		// Non-matches
		{"different domain", "example.com", "other.com", false},
		{"partial domain match", "ample.com", "example.com", false},
		{"superdomain no match", "www.example.com", "example.com", false},
		{"different TLD", "example.com", "example.org", false},

		// Case insensitivity
		{"case insensitive", "Example.COM", "example.com", true},
		{"case insensitive subdomain", ".Example.COM", "www.example.com", true},

		// Edge cases
		{"empty cookie domain", "", "example.com", false},
		{"empty request domain", "example.com", "", false},
		{"both empty", "", "", true},
		{"just dot", ".", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainMatches(tt.cookieDomain, tt.requestDomain)
			if got != tt.want {
				t.Errorf("domainMatches(%q, %q) = %v, want %v",
					tt.cookieDomain, tt.requestDomain, got, tt.want)
			}
		})
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name        string
		cookiePath  string
		requestPath string
		want        bool
	}{
		// Root path
		{"root matches everything", "/", "/anything", true},
		{"root matches root", "/", "/", true},
		{"empty cookie path matches", "", "/anything", true},

		// Exact path match
		{"exact path", "/foo", "/foo", true},
		{"exact path with slash", "/foo/", "/foo/", true},

		// Prefix matches
		{"prefix match with slash", "/foo", "/foo/bar", true},
		{"prefix match trailing slash", "/foo/", "/foo/bar", true},

		// Non-matches
		{"no match different path", "/foo", "/bar", false},
		{"partial match no slash", "/foo", "/foobar", false},
		{"longer cookie path", "/foo/bar", "/foo", false},

		// Empty request path
		{"empty request path", "/foo", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathMatches(tt.cookiePath, tt.requestPath)
			if got != tt.want {
				t.Errorf("pathMatches(%q, %q) = %v, want %v",
					tt.cookiePath, tt.requestPath, got, tt.want)
			}
		})
	}
}
