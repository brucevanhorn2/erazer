package shred

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"syscall"
)

// randomSource returns a stream of random bytes for overwrite passes. A
// nil seed uses crypto/rand (non-reproducible, cryptographically random
// data — the default). A non-nil seed switches to a deterministic
// math/rand stream, letting a user request a reproducible run and making
// tests deterministic.
func randomSource(seed *int64) io.Reader {
	if seed == nil {
		return rand.Reader
	}
	return mrand.New(mrand.NewSource(*seed))
}

// overwritePasses writes `passes` full-length passes of data from src into
// f, fsyncing after each pass. It does not touch f's name, position on
// return, or length — callers handle truncation and deletion separately.
func overwritePasses(f *os.File, size int64, passes int, src io.Reader) (int64, error) {
	var written int64
	for i := 0; i < passes; i++ {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return written, err
		}
		n, err := io.CopyN(f, src, size)
		written += n
		if err != nil {
			return written, err
		}
		if err := f.Sync(); err != nil {
			return written, err
		}
	}
	return written, nil
}

// shredFile overwrites path's content with `passes` full-length passes of
// random data, then truncates it to zero length, renames it to a random
// garbage name in the same directory, and removes it — scrubbing both
// content and filename. It returns the total number of bytes written
// across all passes. The file is opened once, with O_NOFOLLOW: if path has
// been replaced by a symlink since the caller's type check (shredPath's
// os.Lstat), the open fails instead of silently overwriting the link's
// target.
func shredFile(path string, passes int, src io.Reader) (int64, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return 0, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return 0, err
	}

	written, err := overwritePasses(f, info.Size(), passes, src)
	if err != nil {
		f.Close()
		return written, err
	}

	if err := f.Truncate(0); err != nil {
		f.Close()
		return written, err
	}
	if err := f.Close(); err != nil {
		return written, err
	}

	garbagePath, err := renameToGarbage(path)
	if err != nil {
		return written, err
	}
	if err := os.Remove(garbagePath); err != nil {
		return written, err
	}
	return written, nil
}

// renameToGarbage renames path to a random hex name in the same directory
// and returns the new path, so the original filename doesn't survive in
// the filesystem's directory entries any longer than necessary.
func renameToGarbage(path string) (string, error) {
	dir := filepath.Dir(path)
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	newPath := filepath.Join(dir, hex.EncodeToString(buf))
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}
