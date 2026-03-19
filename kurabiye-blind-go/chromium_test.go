package kurabiye

import (
	"testing"
	"time"
)

func TestChromiumEpochToTime(t *testing.T) {
	tests := []struct {
		name      string
		chromium  int64
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantZero  bool
	}{
		{
			name:     "zero value",
			chromium: 0,
			wantZero: true,
		},
		{
			name:      "Unix epoch (1970-01-01)",
			chromium:  11644473600 * 1000000, // seconds * microseconds
			wantYear:  1970,
			wantMonth: time.January,
			wantDay:   1,
		},
		{
			name:      "2024-01-01 00:00:00 UTC",
			chromium:  (11644473600 + 1704067200) * 1000000,
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chromiumEpochToTime(tt.chromium)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("expected zero time, got %v", got)
				}
				return
			}
			gotUTC := got.UTC()
			if gotUTC.Year() != tt.wantYear || gotUTC.Month() != tt.wantMonth || gotUTC.Day() != tt.wantDay {
				t.Errorf("chromiumEpochToTime(%d) = %v, want %d-%02d-%02d",
					tt.chromium, gotUTC, tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

func TestChromiumSameSite(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{-1, ""},
		{0, "None"},
		{1, "Lax"},
		{2, "Strict"},
		{99, ""},
	}

	for _, tt := range tests {
		got := chromiumSameSite(tt.val)
		if got != tt.want {
			t.Errorf("chromiumSameSite(%d) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestFindChromiumCookieDB_NonexistentDir(t *testing.T) {
	path := findChromiumCookieDB("/nonexistent/path/that/does/not/exist")
	if path != "" {
		t.Errorf("expected empty path for nonexistent dir, got %q", path)
	}
}
