//go:build !windows && !linux

package clipboard

import (
	"context"
	"errors"
)

var errUnsupported = errors.New("presse-papiers non pris en charge sur ce système")

func read(context.Context) (content, error) { return content{}, errUnsupported }
func write(context.Context, string) error   { return errUnsupported }
