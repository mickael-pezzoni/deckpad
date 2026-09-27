package files

import (
	"context"
	"strings"

	"golang.org/x/sys/windows"
)

// listDrives parcourt les lettres de lecteur (C:, D:…) : disques internes et USB.
// Les lecteurs réseau et CD sont ignorés (lents ou vides).
func listDrives(_ context.Context) ([]Drive, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, err
	}
	var drives []Drive
	for i := 0; i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		rootPtr, _ := windows.UTF16PtrFromString(root)
		kind := windows.GetDriveType(rootPtr)
		if kind != windows.DRIVE_FIXED && kind != windows.DRIVE_REMOVABLE {
			continue
		}
		var free, total, totalFree uint64
		if err := windows.GetDiskFreeSpaceEx(rootPtr, &free, &total, &totalFree); err != nil {
			continue // lecteur de carte vide, par exemple
		}
		drives = append(drives, Drive{
			Path:      root,
			Name:      volumeName(rootPtr, kind),
			Total:     total,
			Used:      total - totalFree,
			Removable: kind == windows.DRIVE_REMOVABLE || isUSB(root),
		})
	}
	return drives, nil
}

// volumeName renvoie le nom du volume (« Windows », « Jeux »…) comme l'Explorateur.
func volumeName(root *uint16, kind uint32) string {
	buf := make([]uint16, windows.MAX_PATH+1)
	if err := windows.GetVolumeInformation(root, &buf[0], uint32(len(buf)), nil, nil, nil, nil, 0); err == nil {
		if name := strings.TrimSpace(windows.UTF16ToString(buf)); name != "" {
			return name
		}
	}
	if kind == windows.DRIVE_REMOVABLE {
		return "Disque amovible"
	}
	return "Disque local"
}
