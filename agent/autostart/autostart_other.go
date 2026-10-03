//go:build !windows

package autostart

const supported = false

func enabled(string) bool           { return false }
func enable(string, []string) error { return ErrUnsupported }
func disable() error                { return ErrUnsupported }
