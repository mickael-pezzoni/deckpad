package files

import (
	"encoding/binary"
	"unicode/utf16"
)

// lnkTarget lit le chemin visé par un raccourci Windows (.lnk, format « Shell
// Link ») : c'est ce que contient le dossier « Récent ». Seuls les fichiers
// locaux nous intéressent ; un raccourci vers un partage réseau renvoie "".
func lnkTarget(data []byte) string {
	const headerSize = 0x4C
	if len(data) < headerSize || binary.LittleEndian.Uint32(data) != headerSize {
		return ""
	}
	flags := binary.LittleEndian.Uint32(data[0x14:])
	offset := headerSize
	if flags&0x1 != 0 { // HasLinkTargetIDList : on le saute
		if len(data) < offset+2 {
			return ""
		}
		offset += 2 + int(binary.LittleEndian.Uint16(data[offset:]))
	}
	if flags&0x2 == 0 || len(data) < offset+28 { // pas de LinkInfo
		return ""
	}
	info := data[offset:]
	size := int(binary.LittleEndian.Uint32(info))
	if size < 28 || size > len(info) {
		return ""
	}
	info = info[:size]
	header := binary.LittleEndian.Uint32(info[4:])
	if binary.LittleEndian.Uint32(info[8:])&0x1 == 0 { // pas de chemin local
		return ""
	}
	if header >= 0x24 && size >= 36 { // chemins Unicode présents
		base := utf16At(info, binary.LittleEndian.Uint32(info[28:]))
		return base + utf16At(info, binary.LittleEndian.Uint32(info[32:]))
	}
	base := ansiAt(info, binary.LittleEndian.Uint32(info[16:]))
	return base + ansiAt(info, binary.LittleEndian.Uint32(info[24:]))
}

func ansiAt(b []byte, off uint32) string {
	if off == 0 || int(off) >= len(b) {
		return ""
	}
	end := int(off)
	for end < len(b) && b[end] != 0 {
		end++
	}
	return decodeANSI(b[off:end])
}

func utf16At(b []byte, off uint32) string {
	if off == 0 || int(off) >= len(b) {
		return ""
	}
	var u []uint16
	for i := int(off); i+1 < len(b); i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}
