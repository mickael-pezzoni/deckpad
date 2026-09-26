// Package sysinfo fournit les informations générales du PC (page « Infos PC »).
package sysinfo

import (
	"context"
	"os/user"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
)

type Info struct {
	Hostname  string    `json:"hostname"`
	Username  string    `json:"username"`
	OS        string    `json:"os"`
	CPU       string    `json:"cpu"`
	UptimeSec uint64    `json:"uptimeSec"`
	Now       time.Time `json:"now"`
}

func Get(ctx context.Context) (Info, error) {
	h, err := host.InfoWithContext(ctx)
	if err != nil {
		return Info{}, err
	}

	info := Info{
		Hostname:  h.Hostname,
		OS:        h.Platform + " " + h.PlatformVersion,
		UptimeSec: h.Uptime,
		Now:       time.Now(),
	}
	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	}
	if c, err := cpu.InfoWithContext(ctx); err == nil && len(c) > 0 {
		info.CPU = c[0].ModelName
	}
	return info, nil
}
