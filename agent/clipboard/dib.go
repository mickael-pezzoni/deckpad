package clipboard

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math/bits"
)

// Une image copiée sous Windows (capture d'écran, Paint…) arrive souvent en
// « DIB » : un en-tête BITMAPINFOHEADER suivi des pixels, sans l'en-tête de
// fichier .bmp. On la convertit en PNG pour la tablette.

var errDIB = errors.New("image du presse-papiers illisible")

const (
	biRGB       = 0
	biBitfields = 3
	maxPixels   = 64 << 20
)

func isPNG(b []byte) bool {
	return len(b) > 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n"
}

func dibToPNG(dib []byte) ([]byte, error) {
	img, err := decodeDIB(dib)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// decodeDIB lit les images 24 et 32 bits (les seules que Windows fabrique pour
// une capture ou une image copiée depuis une appli).
func decodeDIB(b []byte) (image.Image, error) {
	le := binary.LittleEndian
	if len(b) < 40 {
		return nil, errDIB
	}
	hdr := int(le.Uint32(b[0:]))
	w := int(int32(le.Uint32(b[4:])))
	h := int(int32(le.Uint32(b[8:])))
	bpp := int(le.Uint16(b[14:]))
	compression := le.Uint32(b[16:])
	used := int(le.Uint32(b[32:]))
	if hdr < 40 || hdr > len(b) || w <= 0 || h == 0 || (bpp != 24 && bpp != 32) {
		return nil, errDIB
	}
	topDown := h < 0
	if topDown {
		h = -h
	}
	if w*h > maxPixels {
		return nil, errDIB
	}

	// Position des couleurs dans chaque pixel de 32 bits (par défaut : B, G, R, A).
	masks := [4]uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000}
	offset := hdr + used*4
	switch compression {
	case biRGB:
	case biBitfields:
		if bpp != 32 {
			return nil, errDIB
		}
		m := b[40:]
		if hdr == 40 { // masques placés juste après l'en-tête court
			offset += 12
		}
		if len(m) < 12 {
			return nil, errDIB
		}
		masks[0], masks[1], masks[2] = le.Uint32(m), le.Uint32(m[4:]), le.Uint32(m[8:])
		masks[3] = 0
		if hdr >= 56 {
			masks[3] = le.Uint32(m[12:])
		}
	default:
		return nil, errDIB
	}

	stride := (w*bpp + 31) / 32 * 4
	if offset < 0 || offset+stride*h > len(b) {
		return nil, errDIB
	}
	px := b[offset:]

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	hasAlpha := false
	for y := range h {
		row := px[stride*y:]
		dy := h - 1 - y
		if topDown {
			dy = y
		}
		for x := range w {
			var c color.NRGBA
			if bpp == 24 {
				p := row[x*3:]
				c = color.NRGBA{p[2], p[1], p[0], 0xff}
			} else {
				v := le.Uint32(row[x*4:])
				c = color.NRGBA{channel(v, masks[0]), channel(v, masks[1]), channel(v, masks[2]), channel(v, masks[3])}
				hasAlpha = hasAlpha || c.A != 0
			}
			img.SetNRGBA(x, dy, c)
		}
	}
	// Transparence toute à zéro : le programme ne s'en sert pas, l'image est opaque.
	if bpp == 32 && !hasAlpha {
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xff
		}
	}
	return img, nil
}

// channel extrait une couleur de 8 bits d'un pixel selon son masque.
func channel(v, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	width := bits.OnesCount32(mask)
	c := (v & mask) >> shift
	if width >= 8 {
		return uint8(c >> (width - 8))
	}
	return uint8(c * 255 / (1<<width - 1))
}
