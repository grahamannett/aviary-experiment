//go:build darwin

package kurabiye

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func TestAppleTimestampToTime(t *testing.T) {
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ts := want.Sub(appleEpoch).Seconds()
	got := appleTimestampToTime(ts)
	if !got.Equal(want) {
		t.Errorf("appleTimestampToTime(%v) = %v, want %v", ts, got, want)
	}
}

func TestAppleTimestampZero(t *testing.T) {
	got := appleTimestampToTime(0)
	if !got.IsZero() {
		t.Errorf("appleTimestampToTime(0) should be zero, got %v", got)
	}
}

func TestAppleTimestampNaN(t *testing.T) {
	got := appleTimestampToTime(math.NaN())
	if !got.IsZero() {
		t.Errorf("appleTimestampToTime(NaN) should be zero, got %v", got)
	}
}

// buildTestBinaryCookies builds a minimal Cookies.binarycookies file for testing.
func buildTestBinaryCookies(cookies []testSafariCookie) []byte {
	var buf bytes.Buffer

	// Build cookie records for the single page
	var cookieRecords [][]byte
	for _, c := range cookies {
		cookieRecords = append(cookieRecords, buildCookieRecord(c))
	}

	// Build page
	page := buildPage(cookieRecords)

	// Header: "cook" + num pages (1)
	buf.WriteString("cook")
	binary.Write(&buf, binary.BigEndian, uint32(1))

	// Page size
	binary.Write(&buf, binary.BigEndian, uint32(len(page)))

	// Page data
	buf.Write(page)

	return buf.Bytes()
}

type testSafariCookie struct {
	domain  string
	name    string
	path    string
	value   string
	expiry  float64
	flags   uint32 // 0x1=Secure, 0x4=HttpOnly
}

func buildPage(records [][]byte) []byte {
	var buf bytes.Buffer

	numCookies := uint32(len(records))

	// Calculate offsets: header(4) + numCookies(4) + offsets(4*numCookies) + end(4)
	headerSize := 4 + 4 + 4*numCookies + 4
	offset := headerSize

	// Page header
	binary.Write(&buf, binary.LittleEndian, uint32(0x00000100))
	binary.Write(&buf, binary.LittleEndian, numCookies)

	// Cookie offsets
	offsets := make([]uint32, numCookies)
	for i := range records {
		offsets[i] = uint32(offset)
		offset += uint32(len(records[i]))
	}
	for _, o := range offsets {
		binary.Write(&buf, binary.LittleEndian, o)
	}

	// End of offsets marker
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// Cookie records
	for _, rec := range records {
		buf.Write(rec)
	}

	return buf.Bytes()
}

func buildCookieRecord(c testSafariCookie) []byte {
	// Fixed header: size(4) + flags(4) + padding(4) + urlOff(4) + nameOff(4) + pathOff(4) + valueOff(4) + comment(8) + unknown(4) + expiry(8) + creation(8)
	fixedSize := 4 + 4 + 4 + 4 + 4 + 4 + 4 + 8 + 4 + 8 + 8 // = 56

	// Strings with null terminators
	domainBytes := append([]byte(c.domain), 0)
	nameBytes := append([]byte(c.name), 0)
	pathBytes := append([]byte(c.path), 0)
	valueBytes := append([]byte(c.value), 0)

	totalSize := uint32(fixedSize + len(domainBytes) + len(nameBytes) + len(pathBytes) + len(valueBytes))

	urlOffset := uint32(fixedSize)
	nameOffset := urlOffset + uint32(len(domainBytes))
	pathOffset := nameOffset + uint32(len(nameBytes))
	valueOffset := pathOffset + uint32(len(pathBytes))

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, totalSize)
	binary.Write(&buf, binary.LittleEndian, c.flags)
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // padding
	binary.Write(&buf, binary.LittleEndian, urlOffset)
	binary.Write(&buf, binary.LittleEndian, nameOffset)
	binary.Write(&buf, binary.LittleEndian, pathOffset)
	binary.Write(&buf, binary.LittleEndian, valueOffset)
	binary.Write(&buf, binary.LittleEndian, uint64(0)) // comment (8 bytes)
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // unknown/padding (4 bytes)
	binary.Write(&buf, binary.LittleEndian, c.expiry)
	binary.Write(&buf, binary.LittleEndian, float64(0)) // creation

	buf.Write(domainBytes)
	buf.Write(nameBytes)
	buf.Write(pathBytes)
	buf.Write(valueBytes)

	return buf.Bytes()
}

func TestParseBinaryCookies(t *testing.T) {
	// Expiry: 2030-01-01 in Apple epoch seconds
	expiry := float64(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).Sub(appleEpoch).Seconds())

	data := buildTestBinaryCookies([]testSafariCookie{
		{
			domain: ".example.com",
			name:   "session",
			path:   "/",
			value:  "abc123",
			expiry: expiry,
			flags:  0x1 | 0x4, // Secure + HttpOnly
		},
		{
			domain: ".other.com",
			name:   "token",
			path:   "/api",
			value:  "xyz",
			expiry: expiry,
			flags:  0x1, // Secure only
		},
	})

	cookies, err := parseBinaryCookies(data)
	if err != nil {
		t.Fatalf("parseBinaryCookies error: %v", err)
	}

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != "session" || c.Value != "abc123" || c.Domain != ".example.com" || c.Path != "/" {
		t.Errorf("cookie 0 mismatch: %+v", c)
	}
	if !c.Secure || !c.HTTPOnly {
		t.Errorf("cookie 0 flags wrong: secure=%v httpOnly=%v", c.Secure, c.HTTPOnly)
	}

	c = cookies[1]
	if c.Name != "token" || c.Value != "xyz" || c.Domain != ".other.com" || c.Path != "/api" {
		t.Errorf("cookie 1 mismatch: %+v", c)
	}
	if !c.Secure || c.HTTPOnly {
		t.Errorf("cookie 1 flags wrong: secure=%v httpOnly=%v", c.Secure, c.HTTPOnly)
	}
}

func TestParseBinaryCookiesInvalidMagic(t *testing.T) {
	_, err := parseBinaryCookies([]byte("notcook"))
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestParseBinaryCookiesTooShort(t *testing.T) {
	_, err := parseBinaryCookies([]byte("co"))
	if err == nil {
		t.Error("expected error for short data")
	}
}
