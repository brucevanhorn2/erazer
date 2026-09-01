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

// shredPath destroys path and reports whether path still exists on disk
// afterward — true for a skipped/failed leaf, or for a directory that
// still holds a surviving descendant (even a nested one it never skipped
// itself). The caller uses this to tell removeIfExists whether a
// subsequent ENOTEMPTY is an expected, already-accounted-for outcome or a
// genuine surprise (e.g. an unrelated process writing into the directory
// mid-run) that should be reported as an error.
func shredPath(path string, info os.FileInfo, passes int, src io.Reader, res *Result, skip func(string) bool) bool {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return !removeIfExists(path, res, false)

	case info.IsDir():
		entries, err := os.ReadDir(path)
		if err != nil {
			res.Errors = append(res.Errors, FileError{Path: path, Err: err})
			return true
		}
		survivorRemains := false
		for _, e := range entries {
			childPath := filepath.Join(path, e.Name())
			if skip != nil && skip(childPath) {
				survivorRemains = true
				continue
			}
			childInfo, err := os.Lstat(childPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				res.Errors = append(res.Errors, FileError{Path: childPath, Err: err})
				survivorRemains = true
				continue
			}
			if shredPath(childPath, childInfo, passes, src, res, skip) {
				survivorRemains = true
			}
		}
		return !removeIfExists(path, res, survivorRemains)

	case info.Mode().IsRegular():
		n, err := shredFile(path, passes, src)
		if err != nil {
			res.Errors = append(res.Errors, FileError{Path: path, Err: err})
			return true
		}
		res.FilesShredded++
		res.BytesOverwritten += n
		return false

	default:
		return !removeIfExists(path, res, false)
	}
}

// removeIfExists removes path and reports whether it succeeded (including
// the already-gone case, which counts as success). expectNonEmpty tells it
// whether an ENOTEMPTY failure is an already-accounted-for outcome (an
// intentionally skipped child, or a nested survivor reported up from
// shredPath) rather than a surprise — only in the former case is it
// swallowed instead of reported as a FileError.
func removeIfExists(path string, res *Result, expectNonEmpty bool) bool {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return true
	}
	if expectNonEmpty && errors.Is(err, syscall.ENOTEMPTY) {
		return false
	}
	res.Errors = append(res.Errors, FileError{Path: path, Err: err})
	return false
}
