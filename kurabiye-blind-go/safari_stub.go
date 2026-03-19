//go:build !darwin

package kurabiye

import "fmt"

type safariBrowser struct{}

func newSafari() (*safariBrowser, error) {
	return nil, fmt.Errorf("safari is only supported on macOS")
}

func (b *safariBrowser) name() string {
	return "safari"
}

func (b *safariBrowser) getCookies(host string) ([]Cookie, error) {
	return nil, fmt.Errorf("safari is only supported on macOS")
}
