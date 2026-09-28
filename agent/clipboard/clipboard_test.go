package clipboard

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestShorten(t *testing.T) {
	if s, cut := shorten("abc", 10); s != "abc" || cut {
		t.Fatalf("shorten court : %q %v", s, cut)
	}
	// « é » fait deux octets : on ne le coupe pas en deux.
	s, cut := shorten("aé", 2)
	if s != "a" || !cut {
		t.Fatalf("shorten : %q %v", s, cut)
	}
}

func TestSetTextRefuses(t *testing.T) {
	if err := SetText(t.Context(), ""); err != ErrEmpty {
		t.Fatalf("texte vide : %v", err)
	}
	if err := SetText(t.Context(), strings.Repeat("a", MaxText+1)); err != ErrTooLong {
		t.Fatalf("texte trop long : %v", err)
	}
}

// dib fabrique une image DIB de 2×2 : rouge, vert / bleu, blanc (vue de haut).
func dib(bpp int, bitfields bool, topDown bool) []byte {
	le := binary.LittleEndian
	hdr := make([]byte, 40)
	le.PutUint32(hdr[0:], 40)
	le.PutUint32(hdr[4:], 2)
	h := int32(2)
	if topDown {
		h = -2
	}
	le.PutUint32(hdr[8:], uint32(h))
	le.PutUint16(hdr[12:], 1)
	le.PutUint16(hdr[14:], uint16(bpp))
	b := hdr
	if bitfields {
		le.PutUint32(b[16:], biBitfields)
		b = le.AppendUint32(b, 0x00ff0000)
		b = le.AppendUint32(b, 0x0000ff00)
		b = le.AppendUint32(b, 0x000000ff)
	}
	top := [][3]byte{{255, 0, 0}, {0, 255, 0}}
	bottom := [][3]byte{{0, 0, 255}, {255, 255, 255}}
	rows := [][][3]byte{bottom, top} // de bas en haut
	if topDown {
		rows = [][][3]byte{top, bottom}
	}
	for _, row := range rows {
		n := 0
		for _, p := range row {
			b = append(b, p[2], p[1], p[0])
			n += 3
			if bpp == 32 {
				b = append(b, 0)
				n++
			}
		}
		for n%4 != 0 {
			b = append(b, 0)
			n++
		}
	}
	return b
}

func TestDIBToPNG(t *testing.T) {
	want := [2][2]color.NRGBA{
		{{255, 0, 0, 255}, {0, 255, 0, 255}},
		{{0, 0, 255, 255}, {255, 255, 255, 255}},
	}
	for _, tc := range []struct {
		name      string
		bpp       int
		bitfields bool
		topDown   bool
	}{
		{"24 bits", 24, false, false},
		{"32 bits", 32, false, false},
		{"32 bits masques", 32, true, false},
		{"haut en bas", 24, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := dibToPNG(dib(tc.bpp, tc.bitfields, tc.topDown))
			if err != nil {
				t.Fatal(err)
			}
			if !isPNG(out) {
				t.Fatal("pas un PNG")
			}
			img, err := png.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			for y := range 2 {
				for x := range 2 {
					if got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA); got != want[y][x] {
						t.Errorf("pixel %d,%d : %v, attendu %v", x, y, got, want[y][x])
					}
				}
			}
		})
	}
}

func TestDIBBroken(t *testing.T) {
	good := dib(24, false, false)
	for _, b := range [][]byte{nil, good[:30], good[:len(good)-4]} {
		if _, err := dibToPNG(b); err == nil {
			t.Errorf("image tronquée (%d octets) acceptée", len(b))
		}
	}
}
