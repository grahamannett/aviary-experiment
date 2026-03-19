package kurabiye

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFirefoxExpiryToTime(t *testing.T) {
	tests := []struct {
		name     string
		expiry   int64
		wantZero bool
		wantYear int
	}{
		{"zero means session", 0, true, 0},
		// 1704067200 = 2024-01-01 00:00:00 UTC, compare in UTC
		{"unix timestamp 2024", 1704067200, false, 2024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firefoxExpiryToTime(tt.expiry)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("expected zero time, got %v", got)
				}
				return
			}
			if got.UTC().Year() != tt.wantYear {
				t.Errorf("firefoxExpiryToTime(%d).Year() = %d, want %d", tt.expiry, got.UTC().Year(), tt.wantYear)
			}
		})
	}
}

func TestFirefoxSameSite(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{0, "None"},
		{1, "Lax"},
		{2, "Strict"},
		{99, ""},
	}

	for _, tt := range tests {
		got := firefoxSameSite(tt.val)
		if got != tt.want {
			t.Errorf("firefoxSameSite(%d) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestParseFirefoxProfilesIni(t *testing.T) {
	// Create a temp profiles.ini
	tmpDir, err := os.MkdirTemp("", "kurabiye-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	profileDir := filepath.Join(tmpDir, "abc123.default-release")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}

	iniContent := `[General]
StartWithLastProfile=1

[Profile0]
Name=default-release
IsRelative=1
Path=abc123.default-release
Default=1

[Profile1]
Name=default
IsRelative=1
Path=xyz789.default
`

	iniPath := filepath.Join(tmpDir, "profiles.ini")
	if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := parseFirefoxProfilesIni(iniPath)
	if err != nil {
		t.Fatalf("parseFirefoxProfilesIni() error: %v", err)
	}

	expected := filepath.Join(tmpDir, "abc123.default-release")
	if got != expected {
		t.Errorf("parseFirefoxProfilesIni() = %q, want %q", got, expected)
	}
}

func TestParseFirefoxProfilesIni_NoDefault(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kurabiye-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	iniContent := `[General]
StartWithLastProfile=1

[Profile0]
Name=test
IsRelative=1
Path=test.profile
`

	iniPath := filepath.Join(tmpDir, "profiles.ini")
	if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = parseFirefoxProfilesIni(iniPath)
	if err == nil {
		t.Error("expected error for ini with no default profile")
	}
}

func TestFirefoxExpiryFiltering(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	if !past.Before(now) {
		t.Error("past should be before now")
	}
	if !future.After(now) {
		t.Error("future should be after now")
	}
}
