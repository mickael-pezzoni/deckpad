//go:build !windows && !linux

package media

import (
	"context"
	"errors"
)

var errUnsupported = errors.New("médias non pris en charge sur ce système")

func read(context.Context) (State, error)               { return State{}, errUnsupported }
func control(context.Context, Action) error             { return errUnsupported }
func readCover(context.Context, string) ([]byte, error) { return nil, ErrNoCover }
