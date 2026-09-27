//go:build !windows && !linux

package process

func loadIcon(string, string) (*Icon, error) { return nil, ErrNoIcon }
