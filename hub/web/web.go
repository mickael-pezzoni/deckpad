// Package web embarque la page servie par le serveur central.
package web

import _ "embed"

//go:embed index.html
var Index []byte
