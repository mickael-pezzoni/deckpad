//go:build !windows && !linux

package window

func browsers() []string { return nil }

func fallback(url string) (string, []string) {
	return "open", []string{url}
}
