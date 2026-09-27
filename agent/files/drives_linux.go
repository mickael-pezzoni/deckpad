package files

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// listDrives garde les partitions de vrais disques montées là où l'utilisateur
// range ses fichiers (/, /home, /mnt/…, /media/…), une fois chacune.
func listDrives(ctx context.Context) ([]Drive, error) {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}
	labels := diskLabels()
	seen := map[string]bool{}
	var drives []Drive
	for _, p := range parts {
		if !keepMount(p.Device, p.Mountpoint, p.Fstype) || seen[p.Device] {
			continue
		}
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		seen[p.Device] = true
		drives = append(drives, Drive{
			Path:      p.Mountpoint,
			Name:      driveName(p.Mountpoint, labels[realDevice(p.Device)]),
			Total:     u.Total,
			Used:      u.Used,
			Removable: isUSB(p.Device),
		})
	}
	return drives, nil
}

// keepMount écarte les montages techniques : snaps, /boot, conteneurs, etc.
func keepMount(device, mountpoint, fstype string) bool {
	if !strings.HasPrefix(device, "/dev/") || strings.HasPrefix(device, "/dev/loop") {
		return false
	}
	switch fstype {
	case "squashfs", "iso9660", "udf", "swap":
		return false
	}
	if mountpoint == "/" || mountpoint == "/home" {
		return true
	}
	for _, prefix := range []string{"/home/", "/mnt/", "/media/", "/run/media/", "/srv", "/data"} {
		if strings.HasPrefix(mountpoint, prefix) {
			return true
		}
	}
	return false
}

// driveName : le label du disque, sinon un nom tiré du point de montage.
func driveName(mountpoint, label string) string {
	switch {
	case label != "":
		return label
	case mountpoint == "/":
		return "Système"
	case mountpoint == "/home":
		return "Dossiers personnels"
	default:
		return filepath.Base(mountpoint)
	}
}

// diskLabels lit /dev/disk/by-label : périphérique réel → nom du volume.
func diskLabels() map[string]string {
	labels := map[string]string{}
	const dir = "/dev/disk/by-label"
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		target, err := filepath.EvalSymlinks(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		labels[target] = unescapeLabel(e.Name())
	}
	return labels
}

// udev encode les caractères spéciaux des labels : « Mes\x20jeux ».
func unescapeLabel(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && s[i+1] == 'x' {
			if v, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func realDevice(device string) string {
	if real, err := filepath.EvalSymlinks(device); err == nil {
		return real
	}
	return device
}

// isUSB : le chemin sysfs du périphérique passe par un contrôleur USB.
func isUSB(device string) bool {
	name := filepath.Base(realDevice(device))
	link, err := filepath.EvalSymlinks("/sys/class/block/" + name)
	if err != nil {
		return false
	}
	return strings.Contains(link, "/usb")
}
