// Package webdist embarque l'appli tablette compilée (web/ → dist/) dans l'exe.
package webdist

import "embed"

// Files contient dist/, rempli par `npm run build` dans web/.
//
//go:embed all:dist
var Files embed.FS
