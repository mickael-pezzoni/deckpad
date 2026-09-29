//go:build !windows && !linux

package shortcuts

import "errors"

func keysAvailable() Availability { return Availability{Reason: "Non pris en charge sur ce système"} }

func captureAvailable() Availability { return keysAvailable() }

func capture() error { return errors.New("non pris en charge sur ce système") }

func sendKeys([]string) error { return errors.New("non pris en charge sur ce système") }

func launch(command string) error { return startDetached("sh", "-c", command) }

func defaults() []Shortcut {
	return []Shortcut{
		{ID: "home", Label: "Dossier personnel", Icon: "folder", Color: "orange", Kind: KindOpen, Target: "~"},
	}
}

func typeAvailable() Availability { return keysAvailable() }

func typeText(string) error { return errors.New("non pris en charge sur ce système") }
