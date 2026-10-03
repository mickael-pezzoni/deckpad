// Package icon contient l'icône de deckpad, partagée par l'icône de la zone de
// notification et les notifications affichées sur le PC.
package icon

import _ "embed"

// ICO : format attendu par la zone de notification Windows.
//
//go:embed icon.ico
var ICO []byte

//go:embed icon.png
var PNG []byte
