//go:build !windows && !linux

package tray

// Run : pas d'icône sur ce système, l'agent tourne jusqu'à son arrêt.
func Run(string) { select {} }
