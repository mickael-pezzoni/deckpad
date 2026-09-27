//go:build !windows && !linux

package audio

import (
	"context"
	"errors"
)

var errUnsupported = errors.New("son non pris en charge sur ce système")

func read(context.Context) (State, error)          { return State{}, errUnsupported }
func setVolume(context.Context, target, int) error { return errUnsupported }
func setMute(context.Context, target, bool) error  { return errUnsupported }
func setOutput(context.Context, string) error      { return errUnsupported }
