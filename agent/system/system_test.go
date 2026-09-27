package system

import "testing"

func TestRunRejectsUnknown(t *testing.T) {
	if err := Run("format-c"); err != ErrUnknown {
		t.Errorf("attendu ErrUnknown, obtenu %v", err)
	}
}
