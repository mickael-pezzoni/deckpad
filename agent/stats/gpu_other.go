//go:build !windows && !linux

package stats

func readPlatformGPU() *GPU { return nil }
