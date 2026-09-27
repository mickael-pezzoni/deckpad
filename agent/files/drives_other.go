//go:build !windows && !linux

package files

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
)

func listDrives(ctx context.Context) ([]Drive, error) {
	u, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return nil, err
	}
	return []Drive{{Path: "/", Name: "Système", Total: u.Total, Used: u.Used}}, nil
}
