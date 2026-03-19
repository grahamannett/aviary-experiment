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

type safariBrowser struct{}

func newSafari() (*safariBrowser, error) {
	return &safariBrowser{}, nil
}

func (b *safariBrowser) name() string {
	return "safari"
}

func (b *safariBrowser) getCookies(host string) ([]Cookie, error) {
	var data []byte
	var err error
	for _, path := range safariCookiePaths() {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("reading safari cookies: %w", err)
	}

	allCookies, err := parseBinaryCookies(data)
	if err != nil {
		return nil, fmt.Errorf("parsing safari cookies: %w", err)
	}

	var matched []Cookie
	for _, c := range allCookies {
		if domainMatches(host, c.Domain) {
			matched = append(matched, c)
		}
	}

	return matched, nil
}

// Apple epoch: January 1, 2001 00:00:00 UTC
var appleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

func appleTimestampToTime(seconds float64) time.Time {
	if seconds == 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return time.Time{}
	}
	sec := int64(seconds)
	nsec := int64((seconds - float64(sec)) * 1e9)
	return appleEpoch.Add(time.Duration(sec)*time.Second + time.Duration(nsec)*time.Nanosecond)
}

// parseBinaryCookies parses a Cookies.binarycookies file.
func parseBinaryCookies(data []byte) ([]Cookie, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("file too short")
	}

	// Check magic: "cook"
	if string(data[:4]) != "cook" {
		return nil, fmt.Errorf("invalid magic: %q", data[:4])
	}

	r := bytes.NewReader(data)
	r.Seek(4, 0)

	// Number of pages (big-endian)
	var numPages uint32
	if err := binary.Read(r, binary.BigEndian, &numPages); err != nil {
		return nil, fmt.Errorf("reading page count: %w", err)
	}

	// Page sizes (big-endian)
	pageSizes := make([]uint32, numPages)
	for i := uint32(0); i < numPages; i++ {
		if err := binary.Read(r, binary.BigEndian, &pageSizes[i]); err != nil {
			return nil, fmt.Errorf("reading page size %d: %w", i, err)
		}
	}

	var allCookies []Cookie

	// Read pages sequentially
	for i := uint32(0); i < numPages; i++ {
		pageOffset, _ := r.Seek(0, 1) // current position
		pageData := data[pageOffset : pageOffset+int64(pageSizes[i])]

		cookies, err := parseBinaryCookiePage(pageData)
		if err != nil {
			// Skip malformed pages
			r.Seek(int64(pageSizes[i]), 1)
			continue
		}
		allCookies = append(allCookies, cookies...)

		r.Seek(int64(pageSizes[i]), 1)
	}

	return allCookies, nil
}

func parseBinaryCookiePage(pageData []byte) ([]Cookie, error) {
	if len(pageData) < 8 {
		return nil, fmt.Errorf("page too short")
	}

	pr := bytes.NewReader(pageData)

	// Page header (little-endian): should be 0x00000100
	var pageHeader uint32
	binary.Read(pr, binary.LittleEndian, &pageHeader)
	// 0x00000100 in little-endian = 256
	if pageHeader != 0x00000100 {
		return nil, fmt.Errorf("invalid page header: 0x%08x", pageHeader)
	}

	// Number of cookies in this page
	var numCookies uint32
	binary.Read(pr, binary.LittleEndian, &numCookies)

	// Cookie offsets
	offsets := make([]uint32, numCookies)
	for i := uint32(0); i < numCookies; i++ {
		binary.Read(pr, binary.LittleEndian, &offsets[i])
	}

	var cookies []Cookie
	for _, offset := range offsets {
		if int(offset) >= len(pageData) {
			continue
		}
		c, err := parseBinaryCookieRecord(pageData[offset:])
		if err != nil {
			continue
		}
		cookies = append(cookies, c)
	}

	return cookies, nil
}

func parseBinaryCookieRecord(data []byte) (Cookie, error) {
	if len(data) < 56 { // minimum cookie record size (56-byte fixed header)
		return Cookie{}, fmt.Errorf("cookie record too short")
	}

	r := bytes.NewReader(data)

	var size uint32
	binary.Read(r, binary.LittleEndian, &size)

	var flags uint32
	binary.Read(r, binary.LittleEndian, &flags)

	var padding uint32
	binary.Read(r, binary.LittleEndian, &padding)

	var urlOffset uint32
	binary.Read(r, binary.LittleEndian, &urlOffset)

	var nameOffset uint32
	binary.Read(r, binary.LittleEndian, &nameOffset)

	var pathOffset uint32
	binary.Read(r, binary.LittleEndian, &pathOffset)

	var valueOffset uint32
	binary.Read(r, binary.LittleEndian, &valueOffset)

	// Skip comment (8 bytes) + unknown/padding (4 bytes) = 12 bytes
	r.Seek(12, 1)

	var expiryFloat float64
	binary.Read(r, binary.LittleEndian, &expiryFloat)

	var creationFloat float64
	binary.Read(r, binary.LittleEndian, &creationFloat)

	readString := func(offset uint32) string {
		if int(offset) >= len(data) {
			return ""
		}
		end := int(offset)
		for end < len(data) && data[end] != 0 {
			end++
		}
		return string(data[offset:end])
	}

	domain := readString(urlOffset)
	name := readString(nameOffset)
	path := readString(pathOffset)
	value := readString(valueOffset)

	expires := appleTimestampToTime(expiryFloat)

	secure := flags&0x1 != 0
	httpOnly := flags&0x4 != 0

	return Cookie{
		Name:     name,
		Value:    value,
		Domain:   domain,
		Path:     path,
		Expires:  expires,
		Secure:   secure,
		HTTPOnly: httpOnly,
		SameSite: "", // Safari doesn't expose SameSite in binary format
		Source:   "safari",
	}, nil
}
