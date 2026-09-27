package files

import "testing"

func TestKeepMount(t *testing.T) {
	cases := []struct {
		dev, mount, fs string
		want           bool
	}{
		{"/dev/nvme0n1p2", "/", "ext4", true},
		{"/dev/nvme0n1p3", "/home", "ext4", true},
		{"/dev/sdb1", "/run/media/micka/USB", "vfat", true},
		{"/dev/sdb1", "/media/micka/Jeux", "ntfs3", true},
		{"/dev/sdc1", "/mnt/data", "btrfs", true},
		{"/dev/nvme0n1p1", "/boot/efi", "vfat", false},
		{"/dev/loop3", "/snap/firefox/123", "squashfs", false},
		{"/dev/sr0", "/media/cd", "iso9660", false},
		{"overlay", "/", "overlay", false},
		{"/dev/sda2", "/var/lib/docker", "ext4", false},
	}
	for _, c := range cases {
		if got := keepMount(c.dev, c.mount, c.fs); got != c.want {
			t.Errorf("keepMount(%s, %s) = %v", c.dev, c.mount, got)
		}
	}
}

func TestUnescapeLabel(t *testing.T) {
	if got := unescapeLabel(`Mes\x20jeux`); got != "Mes jeux" {
		t.Errorf("label = %q", got)
	}
}
