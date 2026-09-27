//go:build !windows

package sysinfo

import "context"

func uptime(_ context.Context, hostUptime uint64) uint64 { return hostUptime }
