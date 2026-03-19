//go:build darwin

package kurabiye

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func TestMacAbsoluteTimeToTime(t *testing.T) {
	tests := []struct {
		name     string
		macTime  float64
		wantZero bool
		wantYear int
	}{
		{"zero", 0, true, 0},
		{"NaN", math.NaN(), true, 0},
		{"Inf", math.Inf(1), true, 0},
		// Compare in UTC
		{"2024-01-01", float64(1704067200 - 978307200), false, 2024},
		{"2001-01-01 (epoch)", 0.0, true, 0}, // zero maps to zero time
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := macAbsoluteTimeToTime(tt.macTime)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("expected zero time, got %v", got)
				}
				return
			}
			if got.UTC().Year() != tt.wantYear {
				t.Errorf("macAbsoluteTimeToTime(%f).Year() = %d, want %d",
					tt.macTime, got.UTC().Year(), tt.wantYear)
			}
		})
	}
}

func TestReadNullTerminatedString(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		offset int
		want   string
	}{
		{"normal string", []byte("hello\x00world\x00"), 0, "hello"},
		{"second string", []byte("hello\x00world\x00"), 6, "world"},
		{"empty string", []byte("\x00abc"), 0, ""},
		{"offset past end", []byte("hello"), 10, ""},
		{"negative offset", []byte("hello"), -1, ""},
		{"no null terminator", []byte("hello"), 0, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readNullTerminatedString(tt.data, tt.offset)
			if got != tt.want {
				t.Errorf("readNullTerminatedString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBinaryCookies_InvalidMagic(t *testing.T) {
	_, err := parseBinaryCookies([]byte("nope1234"))
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestParseBinaryCookies_TooSmall(t *testing.T) {
	_, err := parseBinaryCookies([]byte("coo"))
	if err == nil {
		t.Error("expected error for too small file")
	}
}

func TestParseBinaryCookies_ValidMinimal(t *testing.T) {
	// Build a minimal valid binary cookies file with one page and one cookie.
	var buf bytes.Buffer

	// Magic "cook"
	buf.WriteString("cook")

	// Number of pages: 1 (big-endian)
	binary.Write(&buf, binary.BigEndian, uint32(1))

	// We'll build the page first to know its size
	var page bytes.Buffer

	// Page header: 0x00000100 (little-endian)
	binary.Write(&page, binary.LittleEndian, uint32(0x00000100))

	// Number of cookies: 1 (little-endian)
	binary.Write(&page, binary.LittleEndian, uint32(1))

	// Cookie offset: points after the page header (4) + num cookies (4) + offset array (4) = 12
	cookieOffset := uint32(12)
	binary.Write(&page, binary.LittleEndian, cookieOffset)

	// Build cookie record
	var cookie bytes.Buffer

	// We need to calculate string offsets within the cookie record
	// Cookie header: 8 x uint32 (32 bytes) + 2 x float64 (16 bytes) = 48 bytes
	headerSize := 48

	domainStr := ".example.com\x00"
	nameStr := "testname\x00"
	pathStr := "/\x00"
	valueStr := "testvalue\x00"

	urlOffset := uint32(headerSize)
	nameOffset := urlOffset + uint32(len(domainStr))
	pathOffset := nameOffset + uint32(len(nameStr))
	valOffset := pathOffset + uint32(len(pathStr))

	totalSize := uint32(headerSize) + uint32(len(domainStr)+len(nameStr)+len(pathStr)+len(valueStr))

	// Cookie fields
	binary.Write(&cookie, binary.LittleEndian, totalSize)    // size
	binary.Write(&cookie, binary.LittleEndian, uint32(0x01)) // flags (Secure)
	binary.Write(&cookie, binary.LittleEndian, uint32(0))    // unknown
	binary.Write(&cookie, binary.LittleEndian, urlOffset)    // url offset
	binary.Write(&cookie, binary.LittleEndian, nameOffset)   // name offset
	binary.Write(&cookie, binary.LittleEndian, pathOffset)   // path offset
	binary.Write(&cookie, binary.LittleEndian, valOffset)    // value offset
	binary.Write(&cookie, binary.LittleEndian, uint32(0))    // comment

	// Expiry: 2025-01-01 in Mac absolute time
	expiryMac := float64(1735689600 - 978307200)
	binary.Write(&cookie, binary.LittleEndian, expiryMac)  // expiry
	binary.Write(&cookie, binary.LittleEndian, float64(0)) // creation

	// String data
	cookie.WriteString(domainStr)
	cookie.WriteString(nameStr)
	cookie.WriteString(pathStr)
	cookie.WriteString(valueStr)

	page.Write(cookie.Bytes())

	// Page size (big-endian) goes in header
	pageSize := uint32(page.Len())
	binary.Write(&buf, binary.BigEndian, pageSize)

	// Page data
	buf.Write(page.Bytes())

	cookies, err := parseBinaryCookies(buf.Bytes())
	if err != nil {
		t.Fatalf("parseBinaryCookies() error: %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != "testname" {
		t.Errorf("Name = %q, want %q", c.Name, "testname")
	}
	if c.Value != "testvalue" {
		t.Errorf("Value = %q, want %q", c.Value, "testvalue")
	}
	if c.Domain != ".example.com" {
		t.Errorf("Domain = %q, want %q", c.Domain, ".example.com")
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want %q", c.Path, "/")
	}
	if !c.Secure {
		t.Error("expected Secure = true")
	}
	if c.HTTPOnly {
		t.Error("expected HTTPOnly = false")
	}

	// Check expiry year (compare in UTC to avoid timezone issues)
	expectedYear := 2025
	if c.Expires.UTC().Year() != expectedYear {
		t.Errorf("Expires.Year() = %d, want %d (expires=%v)", c.Expires.UTC().Year(), expectedYear, c.Expires.UTC())
	}
}

// Ensure macAbsoluteTimeToTime gives a reasonable result for a known date.
func TestMacAbsoluteTimeKnownDate(t *testing.T) {
	// January 1, 2020 00:00:00 UTC
	// Unix timestamp: 1577836800
	// Mac absolute time: 1577836800 - 978307200 = 599529600
	macTime := float64(599529600)
	got := macAbsoluteTimeToTime(macTime)

	expected := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	gotUTC := got.UTC()
	if gotUTC.Year() != expected.Year() || gotUTC.Month() != expected.Month() || gotUTC.Day() != expected.Day() {
		t.Errorf("macAbsoluteTimeToTime(%f) = %v, want %v", macTime, gotUTC, expected)
	}
}
