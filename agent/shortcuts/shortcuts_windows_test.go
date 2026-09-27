package shortcuts

import (
	"slices"
	"testing"
	"unsafe"
)

func TestEveryKeyHasAWindowsCode(t *testing.T) {
	for _, k := range append(slices.Clone(Modifiers), KeyNames...) {
		if _, ok := vkeys[k]; !ok {
			t.Errorf("pas de code pour %q", k)
		}
	}
}

// SendInput refuse la liste si la taille ne correspond pas à INPUT.
func TestInputSize(t *testing.T) {
	want := uintptr(40)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		want = 28
	}
	if got := unsafe.Sizeof(input{}); got != want {
		t.Errorf("taille %d, attendu %d", got, want)
	}
}
