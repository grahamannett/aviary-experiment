package kurabiye

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProfilesIni(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake profile directory
	profileDir := filepath.Join(tmpDir, "Profiles", "abc123.default-release")
	os.MkdirAll(profileDir, 0o755)

	// Write a profiles.ini
	ini := `[General]
StartWithLastProfile=1

[Profile0]
Name=default-release
IsRelative=1
Path=Profiles/abc123.default-release
Default=1
`
	iniPath := filepath.Join(tmpDir, "profiles.ini")
	os.WriteFile(iniPath, []byte(ini), 0o644)

	got, err := parseProfilesIni(iniPath, tmpDir)
	if err != nil {
		t.Fatalf("parseProfilesIni error: %v", err)
	}

	if got != profileDir {
		t.Errorf("parseProfilesIni = %q, want %q", got, profileDir)
	}
}

func TestParseProfilesIniInstallSection(t *testing.T) {
	tmpDir := t.TempDir()

	profileDir := filepath.Join(tmpDir, "Profiles", "xyz.default-release")
	os.MkdirAll(profileDir, 0o755)

	ini := `[Install308046B0AF4A39CB]
Default=Profiles/xyz.default-release

[Profile0]
Name=default-release
IsRelative=1
Path=Profiles/xyz.default-release
`
	iniPath := filepath.Join(tmpDir, "profiles.ini")
	os.WriteFile(iniPath, []byte(ini), 0o644)

	got, err := parseProfilesIni(iniPath, tmpDir)
	if err != nil {
		t.Fatalf("parseProfilesIni error: %v", err)
	}

	if got != profileDir {
		t.Errorf("parseProfilesIni = %q, want %q", got, profileDir)
	}
}

func TestFindFirefoxProfileByGlob(t *testing.T) {
	tmpDir := t.TempDir()

	profileDir := filepath.Join(tmpDir, "Profiles", "abc.default-release")
	os.MkdirAll(profileDir, 0o755)

	got, err := findFirefoxProfileByGlob(tmpDir)
	if err != nil {
		t.Fatalf("findFirefoxProfileByGlob error: %v", err)
	}

	if got != profileDir {
		t.Errorf("findFirefoxProfileByGlob = %q, want %q", got, profileDir)
	}
}

func TestFindFirefoxProfileByGlobNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := findFirefoxProfileByGlob(tmpDir)
	if err == nil {
		t.Error("expected error when no profile found")
	}
}
