package shred

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Options configures a shred run.
type Options struct {
	// Passes is the number of full-file overwrite passes per file. Values
	// <= 0 fall back to 3.
	Passes int
	// Seed, if non-nil, makes the overwrite data deterministic (see
	// randomSource). Nil means crypto/rand.
	Seed *int64
	// Skip, if non-nil, is consulted for every path encountered while
	// recursing into a directory (not the top-level target itself, which
	// the caller has already chosen to shred). A path for which Skip
	// returns true — file, symlink, or subdirectory — is left completely
	// untouched: not overwritten, not recursed into, not removed. Nil
	// means shred everything under the target, matching prior behavior.
	Skip func(path string) bool
}

// FileError pairs a path with the error encountered while shredding it.
type FileError struct {
	Path string
	Err  error
}

// Result aggregates the outcome of a shred run.
type Result struct {
	FilesShredded    int
	BytesOverwritten int64
	Errors           []FileError
}

// Shred destroys path: a regular file is overwritten and removed, a
// symlink is removed without being followed, a directory has every file
// inside it shredded (recursively) before the now-empty directory tree is
// removed bottom-up, and any other file type (device, socket, FIFO) is
// just unlinked since there's no content to overwrite. A path that no
// longer exists counts as success — the goal (the data not existing) is
// already met.
func Shred(path string, opts Options) Result {
	passes := opts.Passes
	if passes <= 0 {
		passes = 3
	}
	src := randomSource(opts.Seed)

	var res Result
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return res
		}
		res.Errors = append(res.Errors, FileError{Path: path, Err: err})
		return res
	}
	shredPath(path, info, passes, src, &res, opts.Skip)
	return res
}

func shredPath(path string, info os.FileInfo, passes int, src io.Reader, res *Result, skip func(string) bool) {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		removeIfExists(path, res)

	case info.IsDir():
		entries, err := os.ReadDir(path)
		if err != nil {
			res.Errors = append(res.Errors, FileError{Path: path, Err: err})
			return
		}
		for _, e := range entries {
			childPath := filepath.Join(path, e.Name())
			if skip != nil && skip(childPath) {
				continue
			}
			childInfo, err := os.Lstat(childPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				res.Errors = append(res.Errors, FileError{Path: childPath, Err: err})
				continue
			}
			shredPath(childPath, childInfo, passes, src, res, skip)
		}
		removeIfExists(path, res)

	case info.Mode().IsRegular():
		n, err := shredFile(path, passes, src)
		if err != nil {
			res.Errors = append(res.Errors, FileError{Path: path, Err: err})
			return
		}
		res.FilesShredded++
		res.BytesOverwritten += n

	default:
		removeIfExists(path, res)
	}
}

func removeIfExists(path string, res *Result) {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return
	}
	if errors.Is(err, syscall.ENOTEMPTY) {
		return
	}
	res.Errors = append(res.Errors, FileError{Path: path, Err: err})
}
