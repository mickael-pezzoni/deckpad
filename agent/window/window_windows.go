package window

import (
	"os"
	"path/filepath"
)

func browsers() []string {
	var found []string
	for _, root := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LocalAppData")} {
		if root == "" {
			continue
		}
		for _, exe := range []string{`Microsoft\Edge\Application\msedge.exe`, `Google\Chrome\Application\chrome.exe`} {
			p := filepath.Join(root, exe)
			if _, err := os.Stat(p); err == nil {
				found = append(found, p)
			}
		}
	}
	return found
}

func fallback(url string) (string, []string) {
	return "rundll32", []string{"url.dll,FileProtocolHandler", url}
}
