//go:build !windows

package files

import (
	"io/fs"
	"strings"
)

func samePath(a, b string) bool { return a == b }

// isHidden : sous Linux, les fichiers cachés commencent par un point.
func isHidden(name string, _ fs.FileInfo) bool { return strings.HasPrefix(name, ".") }

func pathKey(path string) string { return path }

// hiddenPath : un fichier rangé dans un dossier caché (~/.cache…) ne compte pas.
func hiddenPath(path string) bool {
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// decodeANSI : les raccourcis Windows ne sont lus que sous Windows.
func decodeANSI(b []byte) string { return string(b) }
