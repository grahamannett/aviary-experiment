//go:build darwin

package kurabiye

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"
)

// macAbsoluteTimeEpoch is the Unix timestamp for January 1, 2001 (Mac absolute time epoch).
const macAbsoluteTimeEpoch int64 = 978307200

// extractSafariCookies extracts cookies from Safari's binary cookies file.
func extractSafariCookies(domain string, path string, isSecure bool) ([]Cookie, error) {
	cookiePath := safariCookiesPath()
	if cookiePath == "" {
		return nil, fmt.Errorf("Safari cookies path not available")
	}

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Safari cookies file: %w", err)
	}

	allCookies, err := parseBinaryCookies(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Safari cookies: %w", err)
	}

	// Filter by domain and path
	var cookies []Cookie
	for _, c := range allCookies {
		if !domainMatches(c.Domain, domain) {
			continue
		}
		if !pathMatches(c.Path, path) {
			continue
		}
		cookies = append(cookies, c)
	}

	return cookies, nil
}

// parseBinaryCookies parses Safari's Cookies.binarycookies file format.
func parseBinaryCookies(data []byte) ([]Cookie, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("file too small")
	}

	// Check magic: "cook"
	if string(data[:4]) != "cook" {
		return nil, fmt.Errorf("invalid magic: expected 'cook', got '%s'", string(data[:4]))
	}

	// Number of pages (big-endian uint32)
	numPages := binary.BigEndian.Uint32(data[4:8])

	if len(data) < 8+int(numPages)*4 {
		return nil, fmt.Errorf("file too small for page sizes")
	}

	// Read page sizes (big-endian uint32 each)
	pageSizes := make([]uint32, numPages)
	for i := uint32(0); i < numPages; i++ {
		pageSizes[i] = binary.BigEndian.Uint32(data[8+i*4 : 8+(i+1)*4])
	}

	// Pages start after the header
	offset := 8 + int(numPages)*4

	var allCookies []Cookie

	for i := uint32(0); i < numPages; i++ {
		pageSize := int(pageSizes[i])
		if offset+pageSize > len(data) {
			break // don't read past end
		}

		pageData := data[offset : offset+pageSize]
		cookies, err := parseBinaryCookiePage(pageData)
		if err == nil {
			allCookies = append(allCookies, cookies...)
		}

		offset += pageSize
	}

	return allCookies, nil
}

// parseBinaryCookiePage parses a single page from the binary cookies file.
func parseBinaryCookiePage(pageData []byte) ([]Cookie, error) {
	if len(pageData) < 8 {
		return nil, fmt.Errorf("page too small")
	}

	// Page header: 4 bytes (should be 0x00000100 in little-endian, i.e. 0x00010000 in bytes)
	pageHeader := binary.LittleEndian.Uint32(pageData[:4])
	if pageHeader != 0x00000100 {
		return nil, fmt.Errorf("invalid page header: 0x%08x", pageHeader)
	}

	// Number of cookies in this page (little-endian uint32)
	numCookies := binary.LittleEndian.Uint32(pageData[4:8])

	if len(pageData) < 8+int(numCookies)*4 {
		return nil, fmt.Errorf("page too small for cookie offsets")
	}

	// Cookie offsets (little-endian uint32 each)
	cookieOffsets := make([]uint32, numCookies)
	for i := uint32(0); i < numCookies; i++ {
		cookieOffsets[i] = binary.LittleEndian.Uint32(pageData[8+i*4 : 8+(i+1)*4])
	}

	var cookies []Cookie

	for _, off := range cookieOffsets {
		cookie, err := parseBinaryCookieRecord(pageData, int(off))
		if err != nil {
			continue // skip malformed cookies
		}
		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

// parseBinaryCookieRecord parses a single cookie record from a page.
func parseBinaryCookieRecord(pageData []byte, offset int) (Cookie, error) {
	if offset+48 > len(pageData) {
		return Cookie{}, fmt.Errorf("cookie record too small")
	}

	r := bytes.NewReader(pageData[offset:])

	var (
		size       uint32
		flags      uint32
		unknown    uint32
		urlOffset  uint32
		nameOffset uint32
		pathOffset uint32
		valOffset  uint32
		comment    uint32
	)

	// Read cookie header fields (all little-endian uint32)
	binary.Read(r, binary.LittleEndian, &size)
	binary.Read(r, binary.LittleEndian, &flags)
	binary.Read(r, binary.LittleEndian, &unknown)
	binary.Read(r, binary.LittleEndian, &urlOffset)
	binary.Read(r, binary.LittleEndian, &nameOffset)
	binary.Read(r, binary.LittleEndian, &pathOffset)
	binary.Read(r, binary.LittleEndian, &valOffset)
	binary.Read(r, binary.LittleEndian, &comment)

	// Read expiry and creation dates (float64 little-endian, Mac absolute time)
	var expiryFloat, creationFloat float64
	binary.Read(r, binary.LittleEndian, &expiryFloat)
	binary.Read(r, binary.LittleEndian, &creationFloat)

	// Read null-terminated strings at offsets (relative to start of this cookie record)
	cookieData := pageData[offset:]
	if int(size) > len(cookieData) {
		size = uint32(len(cookieData))
	}
	cookieData = cookieData[:size]

	domain := readNullTerminatedString(cookieData, int(urlOffset))
	name := readNullTerminatedString(cookieData, int(nameOffset))
	path := readNullTerminatedString(cookieData, int(pathOffset))
	value := readNullTerminatedString(cookieData, int(valOffset))

	// Convert Mac absolute time to Go time
	expiry := macAbsoluteTimeToTime(expiryFloat)

	// Parse flags
	isSecure := flags&0x1 != 0
	isHTTPOnly := flags&0x4 != 0

	return Cookie{
		Name:     name,
		Value:    value,
		Domain:   domain,
		Path:     path,
		Expires:  expiry,
		Secure:   isSecure,
		HTTPOnly: isHTTPOnly,
		SameSite: "", // Safari doesn't store SameSite in the binary format
		Source:   "safari",
	}, nil
}

// readNullTerminatedString reads a null-terminated string from data at the given offset.
func readNullTerminatedString(data []byte, offset int) string {
	if offset < 0 || offset >= len(data) {
		return ""
	}

	end := bytes.IndexByte(data[offset:], 0)
	if end == -1 {
		return string(data[offset:])
	}
	return string(data[offset : offset+end])
}

// macAbsoluteTimeToTime converts a Mac absolute time (seconds since 2001-01-01) to Go time.Time.
func macAbsoluteTimeToTime(macTime float64) time.Time {
	if macTime == 0 || math.IsNaN(macTime) || math.IsInf(macTime, 0) {
		return time.Time{}
	}
	unixTime := int64(macTime) + macAbsoluteTimeEpoch
	return time.Unix(unixTime, 0)
}
