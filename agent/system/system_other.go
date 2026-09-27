//go:build !windows && !linux

package system

import "errors"

func run(Action) error { return errors.New("non pris en charge sur ce système") }
