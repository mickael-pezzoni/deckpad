package desktop

import (
	"testing"
	"unsafe"
)

// Windows refuse les structures dont la taille ne correspond pas à la sienne.
func TestStructSizes(t *testing.T) {
	wantInput, wantExec := uintptr(40), uintptr(112)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		wantInput, wantExec = 28, 60
	}
	if got := unsafe.Sizeof(input{}); got != wantInput {
		t.Errorf("INPUT : taille %d, attendu %d", got, wantInput)
	}
	if got := unsafe.Sizeof(shellExecuteInfo{}); got != wantExec {
		t.Errorf("SHELLEXECUTEINFOW : taille %d, attendu %d", got, wantExec)
	}
}
