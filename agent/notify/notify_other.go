//go:build !windows && !linux

package notify

import "errors"

func showSystem(Message) error { return errors.New("pas de notifications sur ce système") }
