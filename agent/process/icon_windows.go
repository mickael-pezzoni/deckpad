package process

import (
	"bytes"
	"image"
	"image/png"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Chaque .exe embarque son icône : on l'extrait à la taille voulue, puis on
// recopie ses pixels (BGRA) dans une image Go.

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	gdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	procPrivateExtractIcons = user32.NewProc("PrivateExtractIconsW")
	procGetIconInfo         = user32.NewProc("GetIconInfo")
	procDestroyIcon         = user32.NewProc("DestroyIcon")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procGetDIBits           = gdi32.NewProc("GetDIBits")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
)

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [256]uint32 // palette, inutilisée en 32 bits mais réservée par sécurité
}

func loadIcon(exePath, _ string) (*Icon, error) {
	img, err := extractIcon(exePath, iconSize)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &Icon{Data: buf.Bytes(), ContentType: "image/png"}, nil
}

func extractIcon(path string, size int) (image.Image, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var hicon uintptr
	n, _, _ := procPrivateExtractIcons.Call(uintptr(unsafe.Pointer(p)), 0,
		uintptr(size), uintptr(size), uintptr(unsafe.Pointer(&hicon)), 0, 1, 0)
	if n == 0 || n == 0xFFFFFFFF || hicon == 0 {
		return nil, ErrNoIcon
	}
	defer procDestroyIcon.Call(hicon)

	var info iconInfo
	if r, _, _ := procGetIconInfo.Call(hicon, uintptr(unsafe.Pointer(&info))); r == 0 {
		return nil, ErrNoIcon
	}
	defer procDeleteObject.Call(info.HbmMask)
	if info.HbmColor == 0 {
		return nil, ErrNoIcon // icône monochrome
	}
	defer procDeleteObject.Call(info.HbmColor)

	color, ok := readBitmap(info.HbmColor, size)
	if !ok {
		return nil, ErrNoIcon
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	hasAlpha := false
	for i := 0; i < size*size; i++ {
		if color[i*4+3] != 0 {
			hasAlpha = true
			break
		}
	}
	// Anciennes icônes sans canal alpha : la transparence vient du masque.
	var mask []byte
	if !hasAlpha {
		mask, _ = readBitmap(info.HbmMask, size)
	}
	for i := 0; i < size*size; i++ {
		b, g, r, a := color[i*4], color[i*4+1], color[i*4+2], color[i*4+3]
		if !hasAlpha {
			a = 255
			if mask != nil && mask[i*4] != 0 {
				a = 0
			}
		}
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = r, g, b, a
	}
	return img, nil
}

// readBitmap lit un bitmap en 32 bits par pixel, de haut en bas.
func readBitmap(hbm uintptr, size int) ([]byte, bool) {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, false
	}
	defer procReleaseDC.Call(0, hdc)

	bi := bitmapInfo{Header: bitmapInfoHeader{
		Width:    int32(size),
		Height:   -int32(size), // valeur négative : lignes de haut en bas
		Planes:   1,
		BitCount: 32,
	}}
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	buf := make([]byte, size*size*4)
	lines, _, _ := procGetDIBits.Call(hdc, hbm, 0, uintptr(size),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0)
	return buf, int(lines) == size
}
