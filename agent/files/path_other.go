//go:build !windows

package files

import (
	"io/fs"
	"strings"
)

func samePath(a, b string) bool { return a == b }

// isHidden : sous Linux, les fichiers cachés commencent par un point.
func isHidden(name string, _ fs.FileInfo) bool { return strings.HasPrefix(name, ".") }
