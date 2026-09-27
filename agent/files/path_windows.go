package files

import (
	"io/fs"
	"strings"
	"syscall"
)

// Les chemins Windows ne tiennent pas compte de la casse.
func samePath(a, b string) bool { return strings.EqualFold(a, b) }

// isHidden masque ce que l'Explorateur cache par défaut (attributs caché ou système).
func isHidden(name string, info fs.FileInfo) bool {
	if a, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return a.FileAttributes&(syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM) != 0
	}
	return false
}
