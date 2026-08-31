package shred

import (
	"os"
	"path/filepath"
	"testing"
)

func withFixtureMounts(t *testing.T, mountsContent string, rotationalByDevice map[string]string) {
	t.Helper()
	dir := t.TempDir()

	mountsPath := filepath.Join(dir, "mounts")
	if err := os.WriteFile(mountsPath, []byte(mountsContent), 0644); err != nil {
		t.Fatalf("write mounts fixture: %v", err)
	}

	blockDir := filepath.Join(dir, "block")
	for dev, value := range rotationalByDevice {
		queueDir := filepath.Join(blockDir, dev, "queue")
		if err := os.MkdirAll(queueDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(queueDir, "rotational"), []byte(value+"\n"), 0644); err != nil {
			t.Fatalf("write rotational fixture: %v", err)
		}
	}

	origMounts, origBlock := procMountsPath, sysBlockPath
	procMountsPath, sysBlockPath = mountsPath, blockDir
	t.Cleanup(func() {
		procMountsPath, sysBlockPath = origMounts, origBlock
	})
}

func TestIsRotational_MatchesLongestMountPrefix(t *testing.T) {
	withFixtureMounts(t,
		"/dev/sda1 / ext4 rw 0 0\n/dev/nvme0n1p1 /mnt/data ext4 rw 0 0\n",
		map[string]string{"sda": "1", "nvme0n1": "0"},
	)

	rotational, ok := IsRotational("/mnt/data/some/file.txt")
	if !ok {
		t.Fatal("expected detection to succeed")
	}
	if rotational {
		t.Fatal("expected /mnt/data to resolve to the non-rotational nvme device")
	}

	rotational, ok = IsRotational("/home/bruce/file.txt")
	if !ok {
		t.Fatal("expected detection to succeed")
	}
	if !rotational {
		t.Fatal("expected the root mount to resolve to the rotational sda device")
	}
}

func TestIsRotational_UnknownWhenSysfsEntryMissing(t *testing.T) {
	withFixtureMounts(t, "/dev/sdz1 / ext4 rw 0 0\n", nil)

	_, ok := IsRotational("/anything")
	if ok {
		t.Fatal("expected detection to fail when the rotational sysfs file is missing")
	}
}

func TestIsRotational_UnknownWhenNoMountMatches(t *testing.T) {
	withFixtureMounts(t, "tmpfs /dev/shm tmpfs rw 0 0\n", nil)

	_, ok := IsRotational("/anything")
	if ok {
		t.Fatal("expected detection to fail when no /dev-backed mount matches")
	}
}
