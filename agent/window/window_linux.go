package window

import "os/exec"

func browsers() []string {
	var found []string
	for _, name := range []string{
		"chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
		"microsoft-edge", "microsoft-edge-stable", "brave-browser",
	} {
		if p, err := exec.LookPath(name); err == nil {
			found = append(found, p)
		}
	}
	return found
}

func fallback(url string) (string, []string) {
	return "xdg-open", []string{url}
}
