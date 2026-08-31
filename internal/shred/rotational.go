package shred

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// procMountsPath and sysBlockPath are package variables (not constants) so
// tests can point them at fixture files instead of the real /proc and
// /sys.
var (
	procMountsPath = "/proc/mounts"
	sysBlockPath   = "/sys/block"
)

// IsRotational reports whether path's filesystem is backed by rotational
// (spinning) storage. ok is false when detection couldn't be completed
// (unusual mount, missing sysfs entry, permission error) — callers should
// treat that as "unknown" and skip any warning rather than guessing.
func IsRotational(path string) (rotational bool, ok bool) {
	device, mountOK := findMountDevice(path)
	if !mountOK {
		return false, false
	}
	base, baseOK := baseBlockDevice(device)
	if !baseOK {
		return false, false
	}
	data, err := os.ReadFile(sysBlockPath + "/" + base + "/queue/rotational")
	if err != nil {
		return false, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, false
	}
	return n == 1, true
}

// findMountDevice returns the device field of the longest mount-point
// prefix of path found in procMountsPath — the mount that actually backs
// path, not just the first line that happens to mention a matching
// device.
func findMountDevice(path string) (device string, ok bool) {
	f, err := os.Open(procMountsPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	bestLen := -1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		dev, mountPoint := fields[0], fields[1]
		if !strings.HasPrefix(dev, "/dev/") {
			continue
		}
		if !hasPathPrefix(path, mountPoint) {
			continue
		}
		if len(mountPoint) > bestLen {
			bestLen = len(mountPoint)
			device = dev
			ok = true
		}
	}
	return device, ok
}

// hasPathPrefix reports whether mountPoint is a path-segment-aligned
// prefix of path (so "/mnt/dat" never matches "/mnt/data/x").
func hasPathPrefix(path, mountPoint string) bool {
	if mountPoint == "/" {
		return true
	}
	if !strings.HasPrefix(path, mountPoint) {
		return false
	}
	return len(path) == len(mountPoint) || path[len(mountPoint)] == '/'
}

// baseBlockDevice strips a partition suffix from a /dev/... device path to
// get the whole-disk name /sys/block entries are keyed by: "/dev/sda1" ->
// "sda", "/dev/nvme0n1p1" -> "nvme0n1".
func baseBlockDevice(device string) (string, bool) {
	name := strings.TrimPrefix(device, "/dev/")
	if name == "" {
		return "", false
	}
	if strings.Contains(name, "nvme") {
		if idx := strings.LastIndex(name, "p"); idx > 0 {
			if _, err := strconv.Atoi(name[idx+1:]); err == nil {
				return name[:idx], true
			}
		}
		return name, true
	}
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == "" {
		return name, true
	}
	return trimmed, true
}
