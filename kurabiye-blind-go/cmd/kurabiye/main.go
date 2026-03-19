package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"kurabiye"
)

func main() {
	urlFlag := flag.String("url", "", "URL to extract cookies for (required)")
	browsersFlag := flag.String("browsers", "", "Comma-separated browsers: chrome,edge,firefox,safari")
	namesFlag := flag.String("names", "", "Comma-separated cookie names to filter")
	modeFlag := flag.String("mode", "merge", "Mode: merge or first")
	headerFlag := flag.Bool("header", false, "Output as Cookie header instead of JSON")
	flag.Parse()

	if *urlFlag == "" {
		// Check environment for browsers/mode
		if envBrowsers := os.Getenv("KURABIYE_BROWSERS"); envBrowsers != "" && *browsersFlag == "" {
			*browsersFlag = envBrowsers
		}
		if envMode := os.Getenv("KURABIYE_MODE"); envMode != "" && *modeFlag == "merge" {
			*modeFlag = envMode
		}
		fmt.Fprintln(os.Stderr, "error: --url is required")
		flag.Usage()
		os.Exit(1)
	}

	// Apply env vars
	if envBrowsers := os.Getenv("KURABIYE_BROWSERS"); envBrowsers != "" && *browsersFlag == "" {
		*browsersFlag = envBrowsers
	}
	if envMode := os.Getenv("KURABIYE_MODE"); envMode != "" && *modeFlag == "merge" {
		*modeFlag = envMode
	}

	opts := kurabiye.GetCookiesOptions{
		URL:  *urlFlag,
		Mode: *modeFlag,
	}

	if *browsersFlag != "" {
		opts.Browsers = strings.Split(*browsersFlag, ",")
	}
	if *namesFlag != "" {
		opts.Names = strings.Split(*namesFlag, ",")
	}

	result, err := kurabiye.GetCookies(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print warnings to stderr
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if *headerFlag {
		header := kurabiye.ToCookieHeader(result.Cookies, true)
		if header != "" {
			fmt.Printf("Cookie: %s\n", header)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
}
