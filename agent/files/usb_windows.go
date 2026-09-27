package files

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Un disque dur USB se présente comme un disque « fixe » : seul le bus du
// périphérique (IOCTL_STORAGE_QUERY_PROPERTY) dit qu'il est branché en USB.
const (
	ioctlStorageQueryProperty = 0x2D1400
	busTypeUSB                = 7
	busTypeOffset             = 28 // STORAGE_DEVICE_DESCRIPTOR.BusType
)

func isUSB(root string) bool {
	path, _ := windows.UTF16PtrFromString(`\\.\` + root[:2])
	h, err := windows.CreateFile(path, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	query := [3]uint32{} // PropertyId = StorageDeviceProperty, QueryType = PropertyStandardQuery
	out := make([]byte, 512)
	var n uint32
	err = windows.DeviceIoControl(h, ioctlStorageQueryProperty,
		(*byte)(unsafe.Pointer(&query[0])), uint32(unsafe.Sizeof(query)),
		&out[0], uint32(len(out)), &n, nil)
	if err != nil || n < busTypeOffset+4 {
		return false
	}
	return binary.LittleEndian.Uint32(out[busTypeOffset:]) == busTypeUSB
}
