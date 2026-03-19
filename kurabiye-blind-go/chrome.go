package kurabiye

func newChrome() (*chromiumBrowser, error) {
	return &chromiumBrowser{
		browserName:    "chrome",
		cookiePaths:    chromeCookiePaths(),
		keychainName:   chromeKeychainName(),
		localStatePath: chromeLocalStatePath(),
	}, nil
}

func newEdge() (*chromiumBrowser, error) {
	return &chromiumBrowser{
		browserName:    "edge",
		cookiePaths:    edgeCookiePaths(),
		keychainName:   edgeKeychainName(),
		localStatePath: edgeLocalStatePath(),
	}, nil
}
