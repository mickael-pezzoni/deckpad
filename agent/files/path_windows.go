package files

import (
	"io/fs"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
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

// pathKey : deux chemins qui ne diffèrent que par la casse sont le même fichier.
func pathKey(path string) string { return strings.ToLower(path) }

// hiddenPath : sous Windows, isHidden sur le fichier suffit.
func hiddenPath(string) bool { return false }

// decodeANSI convertit un texte dans la page de code du système (vieux
// raccourcis sans chemin Unicode).
const cpACP = 0 // page de code ANSI du système

func decodeANSI(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	n, err := windows.MultiByteToWideChar(cpACP, 0, &b[0], int32(len(b)), nil, 0)
	if err != nil || n == 0 {
		return string(b)
	}
	u := make([]uint16, n)
	windows.MultiByteToWideChar(cpACP, 0, &b[0], int32(len(b)), &u[0], n)
	return windows.UTF16ToString(u)
}
