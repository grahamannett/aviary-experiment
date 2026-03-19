package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	kurabiye "github.com/anthropic/kurabiye"
)

func main() {
	urlFlag := flag.String("url", "", "Target URL for cookie extraction (required)")
	browsersFlag := flag.String("browsers", "", "Comma-separated list of browsers: chrome,edge,firefox,safari")
	namesFlag := flag.String("names", "", "Comma-separated list of cookie names to filter")
	modeFlag := flag.String("mode", "", "Mode: 'merge' (default) or 'first'")
	headerFlag := flag.Bool("header", false, "Output as HTTP Cookie header instead of JSON")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: kurabiye [options]\n\n")
		fmt.Fprintf(os.Stderr, "Extract cookies from locally installed web browsers.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  kurabiye --url https://twitter.com --browsers chrome,firefox --names auth_token,ct0\n")
		fmt.Fprintf(os.Stderr, "  kurabiye --url https://twitter.com --browsers chrome --header\n")
	}

	flag.Parse()

	if *urlFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --url is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	opts := kurabiye.GetCookiesOptions{
		URL: *urlFlag,
	}

	// Parse browsers from flag or env
	browserStr := *browsersFlag
	if browserStr == "" {
		browserStr = os.Getenv("KURABIYE_BROWSERS")
	}
	if browserStr != "" {
		opts.Browsers = splitAndTrim(browserStr, ",")
	}

	// Parse names
	if *namesFlag != "" {
		opts.Names = splitAndTrim(*namesFlag, ",")
	}

	// Parse mode from flag or env
	mode := *modeFlag
	if mode == "" {
		mode = os.Getenv("KURABIYE_MODE")
	}
	if mode != "" {
		opts.Mode = mode
	}

	result, err := kurabiye.GetCookies(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print warnings to stderr
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if *headerFlag {
		header := kurabiye.ToCookieHeader(result.Cookies, true)
		if header != "" {
			fmt.Printf("Cookie: %s\n", header)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	}
}

// splitAndTrim splits a string by sep and trims whitespace from each part.
func splitAndTrim(s string, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
