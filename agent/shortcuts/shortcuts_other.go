//go:build !windows && !linux

package shortcuts

import "errors"

func keysSupport() KeysSupport { return KeysSupport{Reason: "Non pris en charge sur ce système"} }

func sendKeys([]string) error { return errors.New("non pris en charge sur ce système") }

func launch(command string) error { return startDetached("sh", "-c", command) }

func open(target string) error { return startDetached("open", target) }

func defaults() []Shortcut {
	return []Shortcut{
		{ID: "home", Label: "Dossier personnel", Icon: "folder", Color: "orange", Kind: KindOpen, Target: "~"},
	}
}
