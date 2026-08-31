# erazer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build erazer, a secure-delete/shredder tool for Linux — a headless CLI mode (`erazer <path>`) and a Bubble Tea TUI mode (`erazer` with no args) — in the same cyberpunk house style as exfil and sneakernet.

**Architecture:** A dependency-free `internal/shred` engine (N-pass overwrite + rename-to-garbage + delete, recursive over directories, best-effort SSD detection) is consumed two ways: synchronously by `internal/headless` for the one-shot CLI path, and via a streaming `Event` channel by `internal/ui`'s Bubble Tea model for the interactive TUI path (Browsing → Confirm → Erasing → Done), with `internal/browse` supplying the same inherited-override multi-select tree sneakernet already uses.

**Tech Stack:** Go 1.26, Bubble Tea, Bubbles (textinput), Lipgloss — same versions as sneakernet (bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0).

**Spec:** `docs/superpowers/specs/2026-08-30-erazer-design.md`

## Global Constraints

- Module path: `github.com/brucevanhorn2/erazer` (matches the real GitHub remote).
- Go version: `1.26.5` in `go.mod` (matches exfil/sneakernet).
- House colors: primary `#B341F5`, secondary `#6E6E6E` (exact hex values, matching exfil/sneakernet).
- About-screen logo gradient: `#00E5FF` → `#B341F5` (matches exfil's About screen exactly).
- Linux only. No Windows/macOS support.
- No BleachBit-style cache/junk cleaning categories, no whole-disk wiping, no settings persistence, no lingo packs — shredder only (see spec's "Out of scope" section).
- License: MIT, copyright "Bruce Van Horn", year 2026 (matches exfil/sneakernet's LICENSE text exactly).

---

## Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `main.go`

**Interfaces:**
- Produces: a buildable `main` package other tasks will replace piece by piece.

- [ ] **Step 1: Create go.mod**

```
module github.com/brucevanhorn2/erazer

go 1.26.5
```

- [ ] **Step 2: Create .gitignore**

```
/erazer
*.test
*.out
/.superpowers/
```

- [ ] **Step 3: Create LICENSE**

```
MIT License

Copyright (c) 2026 Bruce Van Horn

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 4: Create a stub main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("erazer: coming soon")
}
```

- [ ] **Step 5: Verify it builds and run it**

Run: `go build -o erazer . && ./erazer`
Expected: prints `erazer: coming soon`

- [ ] **Step 6: Commit**

```bash
git add go.mod .gitignore LICENSE main.go
git commit -m "Scaffold erazer project"
```

---

## Task 2: internal/browse — selection tree

Ports sneakernet's already-generic, already-tested inherited-override selection tree verbatim (no sneakernet-specific dependencies exist in this file).

**Files:**
- Create: `internal/browse/selection.go`
- Test: `internal/browse/selection_test.go`

**Interfaces:**
- Produces: `browse.Set` (`NewSet()`, `(*Set) Toggle(path)`, `(*Set) Effective(path) bool`, `(*Set) IsExplicit(path) bool`, `(*Set) SelectedRoots() []string`) — consumed by `internal/ui`'s `BrowserPane` and `Model` (Tasks 8-10).

- [ ] **Step 1: Create internal/browse/selection.go**

```go
package browse

import (
	"sort"
	"strings"
)

// Set tracks explicit include/exclude overrides on a path tree. A path's
// effective selection is inherited from the nearest ancestor override,
// defaulting to unselected when no ancestor has one — so selecting a
// folder cascades to everything under it, and a deeper override lets you
// carve out an exception. Unlike a flat per-directory selection map, this
// state is independent of which directory is currently being browsed, so
// it survives navigating away and back.
type Set struct {
	overrides map[string]bool
}

// NewSet returns an empty selection (nothing selected).
func NewSet() *Set {
	return &Set{overrides: make(map[string]bool)}
}

// Toggle flips the effective selection at path by recording an explicit
// override there (the opposite of whatever Effective(path) currently
// returns).
func (s *Set) Toggle(path string) {
	s.overrides[path] = !s.Effective(path)
}

// Effective reports whether path is selected: an explicit override at path
// wins; otherwise it's inherited from the nearest ancestor override; with
// no override anywhere in the chain it's false.
func (s *Set) Effective(path string) bool {
	p := path
	for {
		if v, ok := s.overrides[p]; ok {
			return v
		}
		parent := parentOf(p)
		if parent == p {
			return false
		}
		p = parent
	}
}

// IsExplicit reports whether path itself (not an ancestor) carries an
// override. Used by the UI to distinguish an explicitly-checked entry from
// one that's merely selected because a parent is.
func (s *Set) IsExplicit(path string) bool {
	_, ok := s.overrides[path]
	return ok
}

// SelectedRoots returns the minimal set of paths that are effective-true
// but whose parent is not — the top-level targets a shred run should start
// from. Callers must still consult Effective per-descendant when walking a
// root, since a root can contain deeper excluded exceptions.
func (s *Set) SelectedRoots() []string {
	var roots []string
	for p := range s.overrides {
		if s.Effective(p) && !s.Effective(parentOf(p)) {
			roots = append(roots, p)
		}
	}
	sort.Strings(roots)
	return roots
}

func parentOf(path string) string {
	trimmed := strings.TrimRight(path, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx <= 0 {
		return "/"
	}
	return trimmed[:idx]
}
```

- [ ] **Step 2: Create internal/browse/selection_test.go**

```go
package browse

import "testing"

func TestEffective_DefaultUnselected(t *testing.T) {
	s := NewSet()
	if s.Effective("/home/bruce/Documents") {
		t.Fatal("expected unselected path to be false by default")
	}
}

func TestToggle_SelectsRecursively(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")

	if !s.Effective("/home/bruce/Projects") {
		t.Fatal("expected explicitly toggled path to be selected")
	}
	if !s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected child of selected folder to inherit selection")
	}
	if s.Effective("/home/bruce/Documents") {
		t.Fatal("expected unrelated path to remain unselected")
	}
}

func TestToggle_ChildOverridesParent(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer") // carve out an exception

	if !s.Effective("/home/bruce/Projects") {
		t.Fatal("expected parent to remain selected")
	}
	if s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected explicitly deselected child to be excluded")
	}
	if !s.Effective("/home/bruce/Projects/exfil") {
		t.Fatal("expected sibling to still inherit parent's selection")
	}
}

func TestToggle_Twice_ReselectsChild(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer")
	s.Toggle("/home/bruce/Projects/erazer") // toggle back on

	if !s.Effective("/home/bruce/Projects/erazer") {
		t.Fatal("expected re-toggled child to be selected again")
	}
}

func TestSelectedRoots_FindsTopmostSelectedPaths(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")
	s.Toggle("/home/bruce/Projects/erazer") // excluded exception
	s.Toggle("/home/bruce/Documents/taxes") // a standalone leaf selection

	roots := s.SelectedRoots()
	want := []string{"/home/bruce/Documents/taxes", "/home/bruce/Projects"}
	if len(roots) != len(want) {
		t.Fatalf("got roots %v, want %v", roots, want)
	}
	for i := range want {
		if roots[i] != want[i] {
			t.Fatalf("got roots %v, want %v", roots, want)
		}
	}
}

func TestIsExplicit(t *testing.T) {
	s := NewSet()
	s.Toggle("/home/bruce/Projects")

	if !s.IsExplicit("/home/bruce/Projects") {
		t.Fatal("expected toggled path to be explicit")
	}
	if s.IsExplicit("/home/bruce/Projects/erazer") {
		t.Fatal("expected inherited path to not be explicit")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/browse/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/browse
git commit -m "Add browse.Set inherited-override selection tree"
```

---

## Task 3: internal/shred — core engine (overwrite + recursive shred)

**Files:**
- Create: `internal/shred/overwrite.go`
- Create: `internal/shred/engine.go`
- Test: `internal/shred/overwrite_test.go`
- Test: `internal/shred/engine_test.go`

**Interfaces:**
- Produces: `shred.Options{Passes int; Seed *int64}`, `shred.FileError{Path string; Err error}`, `shred.Result{FilesShredded int; BytesOverwritten int64; Errors []FileError}`, `func Shred(path string, opts Options) Result` — consumed by `internal/headless` (Task 6), `internal/shred/progress.go` (Task 5), and `internal/ui` (Tasks 9-10).

- [ ] **Step 1: Write the failing tests for the overwrite primitives**

`internal/shred/overwrite_test.go`:

```go
package shred

import (
	"bytes"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestOverwritePasses_ChangesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.txt")
	original := []byte("aws_secret_key=ORIGINALVALUE1234567890")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	src := mrand.New(mrand.NewSource(42))
	written, err := overwritePasses(f, int64(len(original)), 2, src)
	if err != nil {
		t.Fatalf("overwritePasses: %v", err)
	}
	if written != int64(len(original))*2 {
		t.Fatalf("got %d bytes written, want %d", written, int64(len(original))*2)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	got := make([]byte, len(original))
	if _, err := io.ReadFull(f, got); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if bytes.Equal(got, original) {
		t.Fatal("expected file content to change after overwrite passes")
	}
}

func TestOverwritePasses_SameSeedSameBytes(t *testing.T) {
	content := []byte("0123456789ABCDEF")
	run := func(seed int64) []byte {
		path := filepath.Join(t.TempDir(), "f.bin")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer f.Close()
		src := mrand.New(mrand.NewSource(seed))
		if _, err := overwritePasses(f, int64(len(content)), 1, src); err != nil {
			t.Fatalf("overwritePasses: %v", err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatalf("seek: %v", err)
		}
		got := make([]byte, len(content))
		if _, err := io.ReadFull(f, got); err != nil {
			t.Fatalf("read back: %v", err)
		}
		return got
	}
	a := run(7)
	b := run(7)
	if !bytes.Equal(a, b) {
		t.Fatal("expected the same seed to produce identical overwrite bytes")
	}
}

func TestShredFile_RemovesOriginalPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.csv")
	if err := os.WriteFile(path, []byte("AKIAABCDEFGHIJKLMNOP"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	n, err := shredFile(path, 2, mrand.New(mrand.NewSource(1)))
	if err != nil {
		t.Fatalf("shredFile: %v", err)
	}
	if n != int64(len("AKIAABCDEFGHIJKLMNOP"))*2 {
		t.Fatalf("got %d bytes written, want %d", n, int64(len("AKIAABCDEFGHIJKLMNOP"))*2)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected original path to no longer exist, got err=%v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/shred/...`
Expected: FAIL — `overwritePasses`/`shredFile` undefined

- [ ] **Step 3: Write internal/shred/overwrite.go**

```go
package shred

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
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
// across all passes.
func shredFile(path string, passes int, src io.Reader) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return 0, err
	}
	written, err := overwritePasses(f, info.Size(), passes, src)
	if err != nil {
		f.Close()
		return written, err
	}
	if err := f.Close(); err != nil {
		return written, err
	}

	if err := os.Truncate(path, 0); err != nil {
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/shred/...`
Expected: PASS

- [ ] **Step 5: Write the failing tests for the recursive engine**

`internal/shred/engine_test.go`:

```go
package shred

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShred_RegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.txt")
	if err := os.WriteFile(path, []byte("AKIA-LEAKED-KEY"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	seed := int64(5)
	res := Shred(path, Options{Passes: 2, Seed: &seed})
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesShredded != 1 {
		t.Fatalf("got FilesShredded=%d, want 1", res.FilesShredded)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone, got err=%v", err)
	}
}

func TestShred_DirectoryRecursion(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "leak")
	nested := filepath.Join(target, "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "a.txt"), []byte("one"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.txt"), []byte("two"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	seed := int64(9)
	res := Shred(target, Options{Passes: 1, Seed: &seed})
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesShredded != 2 {
		t.Fatalf("got FilesShredded=%d, want 2", res.FilesShredded)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be removed, got err=%v", err)
	}
}

func TestShred_SymlinkIsRemovedNotFollowed(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(realFile, []byte("keep me"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	seed := int64(3)
	res := Shred(link, Options{Passes: 1, Seed: &seed})
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("expected symlink to be removed, got err=%v", err)
	}
	if _, err := os.Stat(realFile); err != nil {
		t.Fatalf("expected symlink target to be untouched, got err=%v", err)
	}
}

func TestShred_AlreadyGonePathIsSuccess(t *testing.T) {
	res := Shred(filepath.Join(t.TempDir(), "does-not-exist"), Options{})
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors for an already-missing path, got %v", res.Errors)
	}
	if res.FilesShredded != 0 {
		t.Fatalf("got FilesShredded=%d, want 0", res.FilesShredded)
	}
}

func TestShred_DefaultsPassesWhenNotPositive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	res := Shred(path, Options{Passes: 0})
	if res.FilesShredded != 1 || len(res.Errors) != 0 {
		t.Fatalf("got %+v", res)
	}
}
```

- [ ] **Step 6: Run the tests to verify they fail**

Run: `go test ./internal/shred/...`
Expected: FAIL — `Options`/`Result`/`Shred` undefined

- [ ] **Step 7: Write internal/shred/engine.go**

```go
package shred

import (
	"io"
	"os"
	"path/filepath"
)

// Options configures a shred run.
type Options struct {
	// Passes is the number of full-file overwrite passes per file. Values
	// <= 0 fall back to 3.
	Passes int
	// Seed, if non-nil, makes the overwrite data deterministic (see
	// randomSource). Nil means crypto/rand.
	Seed *int64
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
	shredPath(path, info, passes, src, &res)
	return res
}

func shredPath(path string, info os.FileInfo, passes int, src io.Reader, res *Result) {
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
			childInfo, err := os.Lstat(childPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				res.Errors = append(res.Errors, FileError{Path: childPath, Err: err})
				continue
			}
			shredPath(childPath, childInfo, passes, src, res)
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
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		res.Errors = append(res.Errors, FileError{Path: path, Err: err})
	}
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/shred/...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/shred
git commit -m "Add shred engine: N-pass overwrite, recursive shred, rename-to-garbage"
```

---

## Task 4: internal/shred — rotational (SSD) detection

**Files:**
- Create: `internal/shred/rotational.go`
- Test: `internal/shred/rotational_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func IsRotational(path string) (rotational bool, ok bool)` — consumed by `internal/headless` (Task 6) and `internal/ui`'s Confirm screen (Task 9).

- [ ] **Step 1: Write the failing tests**

`internal/shred/rotational_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/shred/...`
Expected: FAIL — `IsRotational`/`procMountsPath`/`sysBlockPath` undefined

- [ ] **Step 3: Write internal/shred/rotational.go**

```go
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
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/shred/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/shred/rotational.go internal/shred/rotational_test.go
git commit -m "Add best-effort SSD/NVMe rotational detection"
```

---

## Task 5: internal/shred — ShredAll multi-target streaming

**Files:**
- Create: `internal/shred/progress.go`
- Test: `internal/shred/progress_test.go`

**Interfaces:**
- Consumes: `Shred(path, opts) Result` (Task 3).
- Produces: `shred.Event{Path string; Result Result; Done bool}`, `func ShredAll(paths []string, opts Options, events chan<- Event)` — consumed by `internal/headless` (Task 6) and `internal/ui`'s Erasing screen (Task 10).

- [ ] **Step 1: Write the failing test**

`internal/shred/progress_test.go`:

```go
package shred

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShredAll_EmitsPerTargetThenAggregateDone(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("one"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(b, []byte("twotwo"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	seed := int64(11)
	events := make(chan Event, 8)
	ShredAll([]string{a, b}, Options{Passes: 1, Seed: &seed}, events)

	var got []Event
	for ev := range events {
		got = append(got, ev)
	}

	if len(got) != 3 {
		t.Fatalf("got %d events, want 3 (2 targets + done)", len(got))
	}
	if got[0].Path != a || got[0].Done {
		t.Fatalf("event 0 = %+v, want path %q, done=false", got[0], a)
	}
	if got[1].Path != b || got[1].Done {
		t.Fatalf("event 1 = %+v, want path %q, done=false", got[1], b)
	}
	if !got[2].Done {
		t.Fatal("expected the final event to have Done=true")
	}
	if got[2].Result.FilesShredded != 2 {
		t.Fatalf("got aggregate FilesShredded=%d, want 2", got[2].Result.FilesShredded)
	}
	wantBytes := int64(len("one") + len("twotwo"))
	if got[2].Result.BytesOverwritten != wantBytes {
		t.Fatalf("got aggregate BytesOverwritten=%d, want %d", got[2].Result.BytesOverwritten, wantBytes)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/shred/...`
Expected: FAIL — `Event`/`ShredAll` undefined

- [ ] **Step 3: Write internal/shred/progress.go**

```go
package shred

// Event reports progress from ShredAll: one Event is sent after each
// target path finishes (Done is false, Result holds that target's own
// outcome), followed by a final Event with Done set to true and Result
// holding the aggregate across every target.
type Event struct {
	Path   string
	Result Result
	Done   bool
}

// ShredAll shreds each of paths in order, sending one Event per completed
// target followed by a final aggregate Event, then closes events. Targets
// are processed sequentially (not concurrently) so events arrive in a
// predictable order for callers (the TUI's Erasing screen, the headless
// CLI path) to render.
func ShredAll(paths []string, opts Options, events chan<- Event) {
	defer close(events)
	var agg Result
	for _, p := range paths {
		res := Shred(p, opts)
		agg.FilesShredded += res.FilesShredded
		agg.BytesOverwritten += res.BytesOverwritten
		agg.Errors = append(agg.Errors, res.Errors...)
		events <- Event{Path: p, Result: res}
	}
	events <- Event{Done: true, Result: agg}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/shred/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/shred/progress.go internal/shred/progress_test.go
git commit -m "Add ShredAll multi-target progress streaming"
```

---

## Task 6: internal/headless — CLI path

**Files:**
- Create: `internal/headless/run.go`
- Test: `internal/headless/run_test.go`

**Interfaces:**
- Consumes: `shred.IsRotational` (Task 4), `shred.ShredAll`, `shred.Event`, `shred.Options`, `shred.Result` (Tasks 3, 5).
- Produces: `headless.RunArgs{Path string; Passes int; Seed *int64; AssumeYes bool; Stdin io.Reader; Stdout, Stderr io.Writer}`, `func Run(args RunArgs) int` — consumed by `main.go` (Task 11).

- [ ] **Step 1: Write the failing tests**

`internal/headless/run_test.go`:

```go
package headless

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_AssumeYes_ShredsFileAndReportsSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.csv")
	if err := os.WriteFile(path, []byte("AKIA-LEAKED"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	seed := int64(1)
	code := Run(RunArgs{
		Path: path, Passes: 1, Seed: &seed, AssumeYes: true,
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("got exit code %d, want 0; stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone, got err=%v", err)
	}
	if !strings.Contains(stdout.String(), "1 file(s) shredded") {
		t.Fatalf("expected summary in stdout, got %q", stdout.String())
	}
}

func TestRun_DeclineConfirmation_DoesNotShred(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.csv")
	if err := os.WriteFile(path, []byte("keep"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(RunArgs{
		Path: path, Passes: 1, AssumeYes: false,
		Stdin: strings.NewReader("n\n"), Stdout: &stdout, Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("got exit code %d, want 1", code)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to survive a declined confirmation, got err=%v", err)
	}
}

func TestRun_MissingPath_ReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(RunArgs{
		Path: filepath.Join(t.TempDir(), "nope"), AssumeYes: true,
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("got exit code %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Fatal("expected an error message on stderr")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/headless/...`
Expected: FAIL — package/`RunArgs`/`Run` undefined

- [ ] **Step 3: Write internal/headless/run.go**

```go
package headless

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brucevanhorn2/erazer/internal/shred"
)

// RunArgs bundles Run's inputs, including the I/O streams, so tests can
// substitute buffers for stdin/stdout/stderr instead of the real
// terminal.
type RunArgs struct {
	Path      string
	Passes    int
	Seed      *int64
	AssumeYes bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run implements erazer's headless mode: validate the target exists, warn
// if it's on non-rotational storage, confirm unless AssumeYes, shred it,
// and report the outcome. It returns the process exit code: 0 on full
// success, 1 if the target didn't exist, the user declined, or any file
// failed to shred.
func Run(args RunArgs) int {
	if _, err := os.Lstat(args.Path); err != nil {
		fmt.Fprintf(args.Stderr, "erazer: %v\n", err)
		return 1
	}

	if rotational, ok := shred.IsRotational(args.Path); ok && !rotational {
		fmt.Fprintln(args.Stderr, "warning: target appears to be on non-rotational (SSD/NVMe) storage;")
		fmt.Fprintln(args.Stderr, "overwrite-based shredding raises the bar against casual recovery, but wear")
		fmt.Fprintln(args.Stderr, "leveling means it is not a guarantee against determined forensic recovery.")
	}

	if !args.AssumeYes {
		fmt.Fprintf(args.Stdout, "This will permanently and irrecoverably erase %s. Continue? [y/N] ", args.Path)
		reader := bufio.NewReader(args.Stdin)
		line, _ := reader.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			fmt.Fprintln(args.Stdout, "Aborted.")
			return 1
		}
	}

	events := make(chan shred.Event, 4)
	go shred.ShredAll([]string{args.Path}, shred.Options{Passes: args.Passes, Seed: args.Seed}, events)

	var final shred.Result
	for ev := range events {
		if ev.Done {
			final = ev.Result
			continue
		}
		fmt.Fprintf(args.Stdout, "erazed: %s\n", ev.Path)
	}

	fmt.Fprintf(args.Stdout, "%d file(s) shredded, %d bytes overwritten\n", final.FilesShredded, final.BytesOverwritten)
	for _, fe := range final.Errors {
		fmt.Fprintf(args.Stderr, "error: %s: %v\n", fe.Path, fe.Err)
	}
	if len(final.Errors) > 0 {
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/headless/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/headless
git commit -m "Add headless CLI path: validate, confirm, shred, report"
```

---

## Task 7: internal/ui — theme, gradient chrome, about screen

Ports sneakernet's theme/gradient files verbatim (already generic), adds an ERAZE-trigger style pair and a Header style, and adds erazer's own toilet-generated ASCII logo. First task to need Bubble Tea/Lipgloss, so it also pins those dependencies.

**Files:**
- Create: `internal/ui/theme.go`
- Create: `internal/ui/gradient.go`
- Create: `internal/ui/about.go`
- Test: `internal/ui/theme_test.go`
- Test: `internal/ui/gradient_test.go`
- Test: `internal/ui/about_test.go`

**Interfaces:**
- Produces: `ui.Theme` (with `PrimaryColor`, `SecondaryColor`, `MutedPrimaryColor`, `MutedSecondaryColor`, `BrowserDir`, `BrowserFile`, `BrowserSelected`, `Header`, `EraseTrigger`, `EraseTriggerFocused`, `StatusBar`, `StatusKey`, `StatusValue`, `StatusError` — all `lipgloss.Style`/`lipgloss.Color`), `ui.NewTheme() Theme`, `gradientBox(content string, width, height int, from, to lipgloss.Color) string`, `gradientText(s string, from, to lipgloss.Color) string`, `mutedColor(c lipgloss.Color) lipgloss.Color`, `ui.AboutPane`, `ui.NewAboutPane() *AboutPane`, `(*AboutPane) View(theme Theme) string` — consumed by Tasks 8-10.

- [ ] **Step 1: Pin dependencies**

Run: `go get github.com/charmbracelet/lipgloss@v1.1.0 && go get github.com/charmbracelet/bubbletea@v1.3.10`
Expected: `go.mod`/`go.sum` updated with both as direct requirements

- [ ] **Step 2: Create internal/ui/theme.go**

```go
package ui

import "github.com/charmbracelet/lipgloss"

const (
	primaryColorHex   = "#B341F5"
	secondaryColorHex = "#6E6E6E"
)

// Theme holds erazer's static cyberpunk styling — matching exfil's and
// sneakernet's house colors. There's no Settings screen to change these at
// runtime; that's out of scope here.
type Theme struct {
	PrimaryColor        lipgloss.Color
	SecondaryColor      lipgloss.Color
	MutedPrimaryColor   lipgloss.Color
	MutedSecondaryColor lipgloss.Color

	BrowserDir      lipgloss.Style
	BrowserFile     lipgloss.Style
	BrowserSelected lipgloss.Style

	Header lipgloss.Style // section headers, e.g. "Confirm erasure"

	EraseTrigger        lipgloss.Style // the ERAZE button, unfocused
	EraseTriggerFocused lipgloss.Style // the ERAZE button, focused — the "dramatically red" state

	StatusBar   lipgloss.Style
	StatusKey   lipgloss.Style
	StatusValue lipgloss.Style
	StatusError lipgloss.Style
}

func NewTheme() Theme {
	primary := lipgloss.Color(primaryColorHex)
	secondary := lipgloss.Color(secondaryColorHex)
	return Theme{
		PrimaryColor:        primary,
		SecondaryColor:      secondary,
		MutedPrimaryColor:   mutedColor(primary),
		MutedSecondaryColor: mutedColor(secondary),

		BrowserDir: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true),
		BrowserFile: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),
		BrowserSelected: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		Header: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		EraseTrigger: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true),
		EraseTriggerFocused: lipgloss.NewStyle().
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),
		StatusKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true),
		StatusValue: lipgloss.NewStyle().
			Foreground(primary),
		StatusError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")),
	}
}
```

- [ ] **Step 3: Create internal/ui/theme_test.go**

```go
package ui

import "testing"

func TestNewTheme_SetsPrimaryColor(t *testing.T) {
	theme := NewTheme()
	if theme.PrimaryColor != "#B341F5" {
		t.Fatalf("got %v, want #B341F5", theme.PrimaryColor)
	}
	if theme.MutedPrimaryColor == theme.PrimaryColor {
		t.Fatal("expected muted color to differ from the full-strength color")
	}
}
```

- [ ] **Step 4: Create internal/ui/gradient.go**

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// hexToRGB parses a "#RRGGBB" string into its red/green/blue components.
func hexToRGB(hex string) (r, g, b int) {
	_, _ = fmt.Sscanf(strings.TrimPrefix(hex, "#"), "%02x%02x%02x", &r, &g, &b)
	return
}

// lerp linearly interpolates between a and b at position t (0.0-1.0).
func lerp(a, b int, t float64) int {
	return int(float64(a) + t*float64(b-a))
}

// gradientLogo renders text with a horizontal color gradient from `from` to
// `to`, interpolated by each character's column position relative to the
// widest line, so the gradient flows consistently across the whole block
// rather than resetting on each line.
func gradientLogo(text, from, to string) string {
	lines := strings.Split(text, "\n")

	maxWidth := 0
	for _, line := range lines {
		if w := len([]rune(line)); w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth <= 1 {
		return text
	}

	fr, fg, fb := hexToRGB(from)
	tr, tg, tb := hexToRGB(to)

	out := make([]string, len(lines))
	for li, line := range lines {
		var b strings.Builder
		for i, r := range []rune(line) {
			if r == ' ' {
				b.WriteRune(r)
				continue
			}
			t := float64(i) / float64(maxWidth-1)
			hex := fmt.Sprintf("#%02x%02x%02x", lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(string(r)))
		}
		out[li] = b.String()
	}
	return strings.Join(out, "\n")
}

// gradientText applies the same horizontal gradient as gradientLogo to a
// single line of arbitrary text (e.g. a header).
func gradientText(s string, from, to lipgloss.Color) string {
	return gradientLogo(s, string(from), string(to))
}

// mutedColor blends c 50% of the way toward black. Used for unfocused
// panes' gradient endpoints.
func mutedColor(c lipgloss.Color) lipgloss.Color {
	r, g, b := hexToRGB(string(c))
	hex := fmt.Sprintf("#%02x%02x%02x", lerp(r, 0, 0.5), lerp(g, 0, 0.5), lerp(b, 0, 0.5))
	return lipgloss.Color(hex)
}

// gradientBox manually draws a rounded-corner border around content, sized
// to width x height — the *interior* box size, matching lipgloss.Style's
// own Width()/Height() convention. Each border character is colored by its
// position along the box's diagonal, from `from` at the top-left corner to
// `to` at the bottom-right. Width wraps overflowing lines; height is a
// floor, padding shorter content but never truncating taller content.
func gradientBox(content string, width, height int, from, to lipgloss.Color) string {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	wrapped := lipgloss.NewStyle().Width(innerWidth).Render(content)
	lines := strings.Split(wrapped, "\n")

	blankRow := strings.Repeat(" ", innerWidth)
	for len(lines) < height {
		lines = append(lines, blankRow)
	}
	actualHeight := len(lines)

	fr, fg, fb := hexToRGB(string(from))
	tr, tg, tb := hexToRGB(string(to))
	denom := float64(width + actualHeight + 2)

	colorAt := func(x, y int) lipgloss.Color {
		t := float64(x+y) / denom
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		hex := fmt.Sprintf("#%02x%02x%02x", lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t))
		return lipgloss.Color(hex)
	}
	borderChar := func(x, y int, ch string) string {
		return lipgloss.NewStyle().Foreground(colorAt(x, y)).Render(ch)
	}

	rows := make([]string, 0, actualHeight+2)

	var top strings.Builder
	top.WriteString(borderChar(0, 0, "╭"))
	for x := 1; x <= width; x++ {
		top.WriteString(borderChar(x, 0, "─"))
	}
	top.WriteString(borderChar(width+1, 0, "╮"))
	rows = append(rows, top.String())

	for i, line := range lines {
		y := i + 1
		rows = append(rows, borderChar(0, y, "│")+" "+line+" "+borderChar(width+1, y, "│"))
	}

	bottomY := actualHeight + 1
	var bottom strings.Builder
	bottom.WriteString(borderChar(0, bottomY, "╰"))
	for x := 1; x <= width; x++ {
		bottom.WriteString(borderChar(x, bottomY, "─"))
	}
	bottom.WriteString(borderChar(width+1, bottomY, "╯"))
	rows = append(rows, bottom.String())

	return strings.Join(rows, "\n")
}
```

- [ ] **Step 5: Create internal/ui/gradient_test.go**

```go
package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGradientBox_PadsToRequestedHeight(t *testing.T) {
	out := gradientBox("hi", 10, 5, lipgloss.Color("#000000"), lipgloss.Color("#ffffff"))
	lines := strings.Split(out, "\n")
	if len(lines) != 7 { // height + top/bottom border rows
		t.Fatalf("got %d lines, want 7", len(lines))
	}
}

func TestGradientText_ReturnsNonEmptyStringForSingleLine(t *testing.T) {
	out := gradientText("hello", lipgloss.Color("#000000"), lipgloss.Color("#ffffff"))
	if out == "" {
		t.Fatal("expected non-empty rendered text")
	}
}
```

- [ ] **Step 6: Run theme/gradient tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 7: Create internal/ui/about.go**

```go
package ui

import (
	"fmt"
	"strings"
)

// logo is a "bigmono12"-style ASCII rendering of "erazer" (via
// `toilet -f bigmono12 erazer`), colored at render time with a gradient
// instead of baking in ANSI codes here — same approach as exfil's About
// screen.
const logo = `  ░████▒    ██░████   ▒████▓   ████████   ░████▒    ██░████ 
 ░██████▒   ███████   ██████▓  ████████  ░██████▒   ███████ 
 ██▒  ▒██   ███░      █▒  ▒██      ▒██▒  ██▒  ▒██   ███░    
 ████████   ██         ▒█████     ▒██▒   ████████   ██      
 ████████   ██       ░███████    ▒██▒    ████████   ██      
 ██         ██       ██▓░  ██   ▒██▒     ██         ██      
 ███░  ▒█   ██       ██▒  ███  ▒██▒      ███░  ▒█   ██      
 ░███████   ██       ████████  ████████  ░███████   ██      
  ░█████▒   ██        ▓███░██  ████████   ░█████▒   ██      `

// logoFrom and logoTo are the gradient endpoints for the logo: cyan fading
// to purple, matching exfil's About screen exactly.
const (
	logoFrom = "#00E5FF"
	logoTo   = "#B341F5"
	tagline  = "secure delete for people who've had a bad week"
	version  = "dev"
)

// AboutPane renders erazer's About screen: logo, tagline, version, license.
type AboutPane struct {
	Width  int
	Height int
}

func NewAboutPane() *AboutPane {
	return &AboutPane{}
}

func (a *AboutPane) View(theme Theme) string {
	lines := []string{
		gradientLogo(logo, logoFrom, logoTo),
		"",
		theme.BrowserFile.Render(tagline),
		"",
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "Version:")) + version,
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "License:")) + "MIT",
		theme.BrowserDir.Render(fmt.Sprintf("%-10s", "Source:")) + "github.com/brucevanhorn2/erazer",
		"",
		theme.StatusKey.Render("press any key to close"),
	}
	content := strings.Join(lines, "\n")
	return gradientBox(content, a.Width, a.Height-2, theme.PrimaryColor, theme.SecondaryColor)
}
```

- [ ] **Step 8: Create internal/ui/about_test.go**

```go
package ui

import (
	"strings"
	"testing"
)

func TestAboutPane_ViewContainsTagline(t *testing.T) {
	a := NewAboutPane()
	a.Width, a.Height = 60, 20
	theme := NewTheme()
	out := a.View(theme)
	if !strings.Contains(out, "bad week") {
		t.Fatal("expected the tagline text to appear in the rendered output")
	}
}
```

- [ ] **Step 9: Run all internal/ui tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add go.mod go.sum internal/ui
git commit -m "Add theme, gradient chrome, and About screen"
```

---

## Task 8: internal/ui — file browser

Ports sneakernet's single-pane `BrowserPane` verbatim (only the `browse` import path changes) — it already has no sneakernet-specific naming.

**Files:**
- Create: `internal/ui/browser.go`
- Test: `internal/ui/browser_test.go`

**Interfaces:**
- Consumes: `browse.Set` (Task 2), `Theme`/`gradientText`/`gradientBox` (Task 7).
- Produces: `ui.Entry{Name string; IsDir bool}`, `ui.BrowserPane{Cwd string; Entries []Entry; Cursor int; Selection *browse.Set; Width, Height int}`, `ui.NewBrowserPane(start string, sel *browse.Set) *BrowserPane`, `(*BrowserPane) Refresh() error`, `Up()`, `Down()`, `CurrentPath() string`, `Enter() error`, `Back() error`, `ToggleSelect()`, `View(theme Theme) string` — consumed by `internal/ui`'s `Model` (Tasks 9-10).

- [ ] **Step 1: Write the failing tests**

`internal/ui/browser_test.go`:

```go
package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brucevanhorn2/erazer/internal/browse"
)

func TestBrowserPane_SelectionPersistsAcrossNavigation(t *testing.T) {
	home := t.TempDir()
	sub := filepath.Join(home, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if len(b.Entries) == 0 || b.Entries[0].Name != "sub" {
		t.Fatalf("expected first entry to be 'sub', got %+v", b.Entries)
	}
	b.Cursor = 0
	b.ToggleSelect()
	if !sel.Effective(sub) {
		t.Fatal("expected sub to be selected after ToggleSelect")
	}

	if err := b.Enter(); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if b.Cwd != sub {
		t.Fatalf("expected Cwd to be %q, got %q", sub, b.Cwd)
	}
	if err := b.Back(); err != nil {
		t.Fatalf("Back: %v", err)
	}
	if b.Cwd != home {
		t.Fatalf("expected Cwd to be back at %q, got %q", home, b.Cwd)
	}

	if !sel.Effective(sub) {
		t.Fatal("expected selection to survive navigating into and back out of the folder")
	}
}

func TestBrowserPane_CurrentPath(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got := b.CurrentPath(); got != filepath.Join(home, "a.txt") {
		t.Fatalf("got %q, want %q", got, filepath.Join(home, "a.txt"))
	}
}

func TestBrowserPane_UpDown_StaysWithinBounds(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	sel := browse.NewSet()
	b := NewBrowserPane(home, sel)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	b.Up()
	if b.Cursor != 0 {
		t.Fatalf("got Cursor %d, want 0", b.Cursor)
	}
	b.Down()
	if b.Cursor != 0 {
		t.Fatalf("got Cursor %d, want 0", b.Cursor)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ui/...`
Expected: FAIL — `NewBrowserPane`/`BrowserPane` undefined

- [ ] **Step 3: Write internal/ui/browser.go**

```go
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brucevanhorn2/erazer/internal/browse"
)

// Entry is one file or directory listed in the current directory.
type Entry struct {
	Name  string
	IsDir bool
}

// BrowserPane is a single-pane, navigable file listing. Selection lives in
// a shared *browse.Set rather than a per-directory map, so Refresh never
// clears it: a selection made here survives navigating to a different
// directory and back.
type BrowserPane struct {
	Cwd       string
	Entries   []Entry
	Cursor    int
	Selection *browse.Set
	Width     int
	Height    int
	scrollTop int
}

func NewBrowserPane(start string, sel *browse.Set) *BrowserPane {
	return &BrowserPane{Cwd: start, Selection: sel}
}

// Refresh reloads the current directory's listing (directories first, then
// alphabetical), resetting the cursor and scroll position but leaving
// Selection untouched.
func (b *BrowserPane) Refresh() error {
	dirEntries, err := os.ReadDir(b.Cwd)
	if err != nil {
		return err
	}
	entries := make([]Entry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entries = append(entries, Entry{Name: e.Name(), IsDir: e.IsDir()})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	b.Entries = entries
	b.Cursor = 0
	b.scrollTop = 0
	return nil
}

func (b *BrowserPane) Up() {
	if b.Cursor > 0 {
		b.Cursor--
	}
	b.ensureVisible()
}

func (b *BrowserPane) Down() {
	if b.Cursor < len(b.Entries)-1 {
		b.Cursor++
	}
	b.ensureVisible()
}

func (b *BrowserPane) ensureVisible() {
	visibleRows := b.Height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}
	if b.Cursor < b.scrollTop {
		b.scrollTop = b.Cursor
	}
	if b.Cursor >= b.scrollTop+visibleRows {
		b.scrollTop = b.Cursor - visibleRows + 1
	}
}

// CurrentPath returns the absolute path of the entry under the cursor, or
// "" if there is none.
func (b *BrowserPane) CurrentPath() string {
	if b.Cursor < 0 || b.Cursor >= len(b.Entries) {
		return ""
	}
	return filepath.Join(b.Cwd, b.Entries[b.Cursor].Name)
}

func (b *BrowserPane) Enter() error {
	if b.Cursor < 0 || b.Cursor >= len(b.Entries) {
		return nil
	}
	e := b.Entries[b.Cursor]
	if !e.IsDir {
		return nil
	}
	oldCwd := b.Cwd
	b.Cwd = filepath.Join(b.Cwd, e.Name)
	if err := b.Refresh(); err != nil {
		b.Cwd = oldCwd
		return err
	}
	return nil
}

func (b *BrowserPane) Back() error {
	parent := filepath.Dir(b.Cwd)
	if parent == b.Cwd {
		return nil
	}
	oldCwd := b.Cwd
	b.Cwd = parent
	if err := b.Refresh(); err != nil {
		b.Cwd = oldCwd
		return err
	}
	return nil
}

func (b *BrowserPane) ToggleSelect() {
	path := b.CurrentPath()
	if path == "" {
		return
	}
	b.Selection.Toggle(path)
}

func (b *BrowserPane) View(theme Theme) string {
	titleLine := gradientText(fmt.Sprintf(" %s ", b.Cwd), theme.PrimaryColor, theme.SecondaryColor)
	lines := []string{titleLine}

	contentHeight := b.Height - 3
	if contentHeight < 0 {
		contentHeight = 0
	}

	rowsRendered := 0
	for i := b.scrollTop; i < len(b.Entries) && i < b.scrollTop+contentHeight; i++ {
		e := b.Entries[i]
		path := filepath.Join(b.Cwd, e.Name)
		effective := b.Selection.Effective(path)
		explicit := b.Selection.IsExplicit(path)

		cursorMark := " "
		if i == b.Cursor {
			cursorMark = "►"
		}
		selectMark := " "
		switch {
		case effective && explicit:
			selectMark = "☑"
		case effective:
			selectMark = "·"
		}
		marker := cursorMark + selectMark + " "

		style := theme.BrowserFile
		if e.IsDir {
			style = theme.BrowserDir
		}
		if effective {
			style = theme.BrowserSelected
		}

		line := marker + style.Render(e.Name)
		if e.IsDir {
			line += "/"
		}
		lines = append(lines, line)
		rowsRendered++
	}
	for ; rowsRendered < contentHeight; rowsRendered++ {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return gradientBox(content, b.Width, b.Height-2, theme.PrimaryColor, theme.SecondaryColor)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/browser.go internal/ui/browser_test.go
git commit -m "Add single-pane file browser"
```

---

## Task 9: internal/ui — Model, Browsing/About/Confirm/Done screens

Builds the Bubble Tea `Model` and its screen state machine, ending with a **synchronous** erase on the Confirm screen's trigger (no animation yet — that's Task 10's job). This is the correctness-first pass; Task 10 layers the "derez" visual on top without changing what actually happens.

**Files:**
- Create: `internal/ui/app.go`
- Create: `internal/ui/view.go`
- Test: `internal/ui/app_test.go`

**Interfaces:**
- Consumes: `browse.NewSet`/`browse.Set` (Task 2), `shred.Shred`/`shred.Options`/`shred.Result`/`shred.IsRotational` (Tasks 3-4), `NewBrowserPane`/`BrowserPane` (Task 8), `NewAboutPane`/`AboutPane`, `NewTheme`/`Theme` (Task 7).
- Produces: `ui.Model` implementing `tea.Model` (`Init`, `Update`, `View`), `ui.NewModel(start string) Model` — consumed by `main.go` (Task 11). Task 10 will modify this file's `startErase` method and add a new screen.

- [ ] **Step 1: Pin the bubbles dependency**

Run: `go get github.com/charmbracelet/bubbles@v1.0.0`
Expected: `go.mod`/`go.sum` updated

- [ ] **Step 2: Write the failing tests**

`internal/ui/app_test.go`:

```go
package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_StartsOnBrowsingScreen(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(dir)
	if m.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", m.screen)
	}
}

func TestHandleBrowsingKey_ENoSelectionStaysOnBrowsing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)

	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := updated.(Model)
	if got.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", got.screen)
	}
}

func TestHandleBrowsingKey_ESelectionMovesToConfirm(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.Cursor = 0
	m.browser.ToggleSelect()

	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := updated.(Model)
	if got.screen != screenConfirm {
		t.Fatalf("got screen %v, want screenConfirm", got.screen)
	}
	if len(got.targets) != 1 {
		t.Fatalf("got %d targets, want 1", len(got.targets))
	}
}

func TestHandleBrowsingKey_QuestionMarkOpensAbout(t *testing.T) {
	dir := t.TempDir()
	m := NewModel(dir)
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	got := updated.(Model)
	if got.screen != screenAbout {
		t.Fatalf("got screen %v, want screenAbout", got.screen)
	}
}

func TestConfirmScreen_EscReturnsToBrowsing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.screen != screenBrowsing {
		t.Fatalf("got screen %v, want screenBrowsing", got.screen)
	}
}

func TestConfirmScreen_TabCyclesFocus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	if m.confirmFocus != 0 {
		t.Fatalf("got initial confirmFocus %d, want 0", m.confirmFocus)
	}
	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.confirmFocus != 1 {
		t.Fatalf("got confirmFocus %d after one tab, want 1", m.confirmFocus)
	}
	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.confirmFocus != 2 {
		t.Fatalf("got confirmFocus %d after two tabs, want 2", m.confirmFocus)
	}
}

func TestConfirmScreen_InvalidPassesShowsErrorAndDoesNotErase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)

	m.passesInput.SetValue("not-a-number")
	m.confirmFocus = 2

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.screen != screenConfirm {
		t.Fatalf("got screen %v, want to stay on screenConfirm", got.screen)
	}
	if got.confirmErr == "" {
		t.Fatal("expected a validation error message")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to survive a failed validation, got err=%v", err)
	}
}

func TestConfirmScreen_TriggerErasesSelectedFileAndReachesDone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("secret"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	m.confirmFocus = 2

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.screen != screenDone {
		t.Fatalf("got screen %v, want screenDone", got.screen)
	}
	if got.result.FilesShredded != 1 {
		t.Fatalf("got FilesShredded=%d, want 1", got.result.FilesShredded)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone, got err=%v", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./internal/ui/...`
Expected: FAIL — `Model`/`NewModel`/`screenBrowsing` etc. undefined

- [ ] **Step 4: Write internal/ui/app.go**

```go
package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/brucevanhorn2/erazer/internal/browse"
	"github.com/brucevanhorn2/erazer/internal/shred"
)

type screen int

const (
	screenBrowsing screen = iota
	screenAbout
	screenConfirm
	screenDone
)

const defaultPasses = 3

// Model is erazer's whole application state: which screen is active, the
// shared selection tree, and per-screen data.
type Model struct {
	screen screen
	theme  Theme
	width  int
	height int

	selection *browse.Set
	browser   *BrowserPane
	about     *AboutPane

	passesInput  textinput.Model
	seedInput    textinput.Model
	confirmFocus int // 0 = passes field, 1 = seed field, 2 = ERAZE trigger
	confirmErr   string
	rotational   bool
	rotationalOK bool

	targets []string
	result  shred.Result
	doneErr string

	quitting bool
}

// NewModel builds the initial model, starting the browser at start (the
// entrypoint passes the user's home directory).
func NewModel(start string) Model {
	sel := browse.NewSet()
	browser := NewBrowserPane(start, sel)
	_ = browser.Refresh()

	passesInput := textinput.New()
	passesInput.Placeholder = strconv.Itoa(defaultPasses)
	passesInput.CharLimit = 3
	passesInput.SetValue(strconv.Itoa(defaultPasses))

	seedInput := textinput.New()
	seedInput.Placeholder = "blank = crypto/rand"
	seedInput.CharLimit = 20

	return Model{
		screen:      screenBrowsing,
		theme:       NewTheme(),
		selection:   sel,
		browser:     browser,
		about:       NewAboutPane(),
		passesInput: passesInput,
		seedInput:   seedInput,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.browser.Width = msg.Width
		m.browser.Height = msg.Height - 2
		m.about.Width = msg.Width
		m.about.Height = msg.Height - 2
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		switch m.screen {
		case screenBrowsing:
			return m.handleBrowsingKey(msg)
		case screenAbout:
			return m.handleAboutKey(msg)
		case screenConfirm:
			return m.handleConfirmKey(msg)
		case screenDone:
			return m.handleDoneKey(msg)
		}
	}
	return m, nil
}

func (m Model) handleBrowsingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		m.browser.Up()
	case "down", "j":
		m.browser.Down()
	case "enter", "l", "right":
		if err := m.browser.Enter(); err != nil {
			m.doneErr = err.Error()
			m.screen = screenDone
		}
	case "backspace", "h", "left":
		if err := m.browser.Back(); err != nil {
			m.doneErr = err.Error()
			m.screen = screenDone
		}
	case " ":
		m.browser.ToggleSelect()
	case "?":
		m.screen = screenAbout
	case "e":
		m.targets = m.selection.SelectedRoots()
		if len(m.targets) == 0 {
			return m, nil
		}
		m.rotational, m.rotationalOK = false, false
		for _, target := range m.targets {
			r, ok := shred.IsRotational(target)
			if !ok {
				continue
			}
			m.rotationalOK = true
			if !r {
				m.rotational = false
				break
			}
			m.rotational = true
		}
		m.confirmFocus = 0
		m.confirmErr = ""
		m.passesInput.Focus()
		m.seedInput.Blur()
		m.screen = screenConfirm
	}
	return m, nil
}

func (m Model) handleAboutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.screen = screenBrowsing
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenBrowsing
		return m, nil
	case "tab", "down":
		m.confirmFocus = (m.confirmFocus + 1) % 3
		m.syncConfirmFocus()
		return m, nil
	case "shift+tab", "up":
		m.confirmFocus = (m.confirmFocus + 2) % 3
		m.syncConfirmFocus()
		return m, nil
	case "enter":
		if m.confirmFocus == 2 {
			return m.startErase()
		}
		m.confirmFocus = (m.confirmFocus + 1) % 3
		m.syncConfirmFocus()
		return m, nil
	}
	var cmd tea.Cmd
	switch m.confirmFocus {
	case 0:
		m.passesInput, cmd = m.passesInput.Update(msg)
	case 1:
		m.seedInput, cmd = m.seedInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) syncConfirmFocus() {
	if m.confirmFocus == 0 {
		m.passesInput.Focus()
	} else {
		m.passesInput.Blur()
	}
	if m.confirmFocus == 1 {
		m.seedInput.Focus()
	} else {
		m.seedInput.Blur()
	}
}

// parseConfirmSettings validates the passes/seed fields, returning the
// parsed shred.Options or an error message fit to show on the Confirm
// screen.
func (m Model) parseConfirmSettings() (shred.Options, string) {
	passesStr := strings.TrimSpace(m.passesInput.Value())
	if passesStr == "" {
		passesStr = strconv.Itoa(defaultPasses)
	}
	passes, err := strconv.Atoi(passesStr)
	if err != nil || passes <= 0 {
		return shred.Options{}, "passes must be a positive whole number"
	}

	seedStr := strings.TrimSpace(m.seedInput.Value())
	var seed *int64
	if seedStr != "" {
		v, err := strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			return shred.Options{}, "seed must be a whole number"
		}
		seed = &v
	}
	return shred.Options{Passes: passes, Seed: seed}, ""
}

// startErase runs the shred synchronously and moves straight to Done. Task
// 10 replaces this body with an async, animated version — the Confirm ->
// Done contract (m.result, m.screen) stays the same either way.
func (m Model) startErase() (tea.Model, tea.Cmd) {
	opts, errMsg := m.parseConfirmSettings()
	if errMsg != "" {
		m.confirmErr = errMsg
		return m, nil
	}
	m.result = shred.Result{}
	for _, target := range m.targets {
		res := shred.Shred(target, opts)
		m.result.FilesShredded += res.FilesShredded
		m.result.BytesOverwritten += res.BytesOverwritten
		m.result.Errors = append(m.result.Errors, res.Errors...)
	}
	m.screen = screenDone
	return m, nil
}

func (m Model) handleDoneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.quitting = true
	return m, tea.Quit
}
```

- [ ] **Step 5: Write internal/ui/view.go**

```go
package ui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenAbout:
		return m.about.View(m.theme)
	case screenConfirm:
		return m.confirmView()
	case screenDone:
		return m.doneView()
	default:
		return m.browsingView()
	}
}

func (m Model) browsingView() string {
	hint := m.theme.StatusBar.Render(
		"↑/↓ move  enter open  space select  e erase selected  ? about  q quit",
	)
	return m.browser.View(m.theme) + "\n" + hint
}

func (m Model) confirmView() string {
	var b strings.Builder
	b.WriteString(m.theme.Header.Render("Confirm erasure"))
	b.WriteString("\n\n")
	b.WriteString("Targets:\n")
	for _, t := range m.targets {
		b.WriteString("  " + t + "\n")
	}
	b.WriteString("\n")
	if m.rotationalOK && !m.rotational {
		b.WriteString(m.theme.StatusError.Render(
			"warning: at least one target is on non-rotational (SSD/NVMe) storage;\n"+
				"overwrite shredding is not a guarantee against forensic recovery on flash media.") + "\n\n")
	}
	b.WriteString(m.theme.StatusKey.Render("Passes: ") + m.passesInput.View() + "\n")
	b.WriteString(m.theme.StatusKey.Render("Seed:   ") + m.seedInput.View() + "\n\n")

	trigger := "[ ERAZE ]"
	if m.confirmFocus == 2 {
		trigger = m.theme.EraseTriggerFocused.Render(trigger)
	} else {
		trigger = m.theme.EraseTrigger.Render(trigger)
	}
	b.WriteString(trigger + "\n")

	if m.confirmErr != "" {
		b.WriteString("\n" + m.theme.StatusError.Render(m.confirmErr) + "\n")
	}
	b.WriteString("\n" + m.theme.StatusBar.Render("tab/shift+tab move  enter confirm  esc back"))
	return b.String()
}

func (m Model) doneView() string {
	var b strings.Builder
	b.WriteString(m.theme.StatusKey.Render("Done.") + "\n\n")
	if m.doneErr != "" {
		b.WriteString(m.theme.StatusError.Render(m.doneErr) + "\n")
	} else {
		b.WriteString(fmt.Sprintf("%d file(s) shredded, %d bytes overwritten\n", m.result.FilesShredded, m.result.BytesOverwritten))
		for _, e := range m.result.Errors {
			b.WriteString(m.theme.StatusError.Render(fmt.Sprintf("error: %s: %v", e.Path, e.Err)) + "\n")
		}
	}
	b.WriteString("\n" + m.theme.StatusBar.Render("press any key to quit"))
	return b.String()
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/ui/app.go internal/ui/view.go internal/ui/app_test.go
git commit -m "Add Model with Browsing/About/Confirm/Done screens (synchronous erase)"
```

---

## Task 10: internal/ui — Erasing screen and derez animation

Replaces `startErase`'s synchronous loop with the async `shred.ShredAll` + channel "subscription" pattern (same technique exfil/sneakernet use for transfer/backup progress), and adds the Erasing screen: each target's filename glitches through the gradient colors while it's shredded, gated so the UI never claims a target is done before the real `shred.Event` for it has arrived.

**Files:**
- Create: `internal/ui/dissolve.go`
- Modify: `internal/ui/app.go` (add `screenErasing`, replace `startErase`, add `eraseEventMsg`/`dissolveTickMsg` handling)
- Modify: `internal/ui/view.go` (add the Erasing screen's rendering)
- Test: `internal/ui/dissolve_test.go`
- Modify: `internal/ui/app_test.go` (`TestConfirmScreen_TriggerErasesSelectedFileAndReachesDone` now expects `screenErasing`, not `screenDone`; add the new erasing-flow test below)

**Interfaces:**
- Consumes: `shred.ShredAll`, `shred.Event` (Task 5).
- Produces: `dissolveText(name string, frame, frames int) string`, `dissolveFrameCount` constant, `eraseEventMsg`, `dissolveTickMsg`, `screenErasing` — internal to `internal/ui`, consumed only by `main.go`'s use of `Model` as a whole (Task 11).

- [ ] **Step 1: Write the failing test for the pure dissolve function**

`internal/ui/dissolve_test.go`:

```go
package ui

import "testing"

func TestDissolveText_FrameZeroIsUnchanged(t *testing.T) {
	got := dissolveText("secret.csv", 0, 6)
	if got != "secret.csv" {
		t.Fatalf("got %q, want unchanged string at frame 0", got)
	}
}

func TestDissolveText_FinalFrameFullyGlitched(t *testing.T) {
	original := "secret.csv"
	got := dissolveText(original, 6, 6)
	if got == original {
		t.Fatal("expected the final frame to differ from the original text")
	}
	if len([]rune(got)) != len([]rune(original)) {
		t.Fatalf("got length %d, want %d", len([]rune(got)), len([]rune(original)))
	}
}

func TestDissolveText_ProgressesPartway(t *testing.T) {
	original := "secret.csv"
	full := dissolveText(original, 6, 6)
	half := dissolveText(original, 3, 6)
	if half == original || half == full {
		t.Fatal("expected a partial frame to differ from both the original and the final frame")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/ui/...`
Expected: FAIL — `dissolveText` undefined

- [ ] **Step 3: Write internal/ui/dissolve.go**

```go
package ui

import "time"

const (
	dissolveFrameCount = 6
	dissolveInterval   = 120 * time.Millisecond
)

// glitchChars reuses the block-shading characters from the About screen's
// logo, so the erasing animation feels visually consistent with the rest
// of erazer's chrome.
var glitchChars = []rune("░▒▓█#%&@$")

// dissolveText renders name with an increasing fraction of its characters
// replaced by glitch characters as frame advances toward frames,
// simulating a "derez" effect — a cheap, deterministic, campy stand-in for
// a real dissolve animation. At frame >= frames the whole string is
// glitched.
func dissolveText(name string, frame, frames int) string {
	runes := []rune(name)
	if frames <= 0 {
		frames = 1
	}
	cutoff := len(runes) * frame / frames
	if cutoff > len(runes) {
		cutoff = len(runes)
	}
	out := make([]rune, len(runes))
	for i, r := range runes {
		if i < cutoff {
			out[i] = glitchChars[(i+frame)%len(glitchChars)]
		} else {
			out[i] = r
		}
	}
	return string(out)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 5: Write the failing tests for the erasing state machine**

Update `internal/ui/app_test.go`: replace `TestConfirmScreen_TriggerErasesSelectedFileAndReachesDone`'s body and add a new test for the tick/event gating, using direct `Update()` calls (not `handle*Key`) since `eraseEventMsg`/`dissolveTickMsg` aren't key messages — matching how sneakernet's own tests drive non-key messages straight through `Update`.

```go
func TestConfirmScreen_TriggerStartsErasingScreen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("secret"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := NewModel(dir)
	m.browser.ToggleSelect()
	updated, _ := m.handleBrowsingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	m.confirmFocus = 2

	updated, _ = m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.screen != screenErasing {
		t.Fatalf("got screen %v, want screenErasing", got.screen)
	}
	if len(got.targets) != 1 || got.targets[0] != path {
		t.Fatalf("got targets %v, want [%s]", got.targets, path)
	}
	if got.eventsCh == nil {
		t.Fatal("expected eventsCh to be set")
	}

	// Drain the real shred so the temp dir cleans up without a race.
	for range got.eventsCh {
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to eventually be gone, got err=%v", err)
	}
}

func TestErasingScreen_TicksWaitForEventBeforeAdvancing(t *testing.T) {
	m := Model{
		screen:  screenErasing,
		targets: []string{"/tmp/a", "/tmp/b"},
	}

	for i := 0; i < dissolveFrameCount+3; i++ {
		updated, _ := m.Update(dissolveTickMsg{})
		m = updated.(Model)
	}
	if m.dissolveFrame != dissolveFrameCount {
		t.Fatalf("got dissolveFrame %d, want it capped at %d", m.dissolveFrame, dissolveFrameCount)
	}
	if m.targetIdx != 0 {
		t.Fatal("expected targetIdx to hold at 0 until the real event for target 0 arrives")
	}

	updated, _ := m.Update(eraseEventMsg{Path: "/tmp/a", Result: shred.Result{FilesShredded: 1}})
	m = updated.(Model)
	if !m.targetDone {
		t.Fatal("expected targetDone once the event for the current target arrives")
	}

	updated, _ = m.Update(dissolveTickMsg{})
	m = updated.(Model)
	if m.targetIdx != 1 {
		t.Fatalf("got targetIdx %d, want 1", m.targetIdx)
	}
	if m.dissolveFrame != 0 {
		t.Fatalf("got dissolveFrame %d, want reset to 0 for the new target", m.dissolveFrame)
	}
	if m.targetDone {
		t.Fatal("expected targetDone to reset for the new target")
	}

	updated, _ = m.Update(eraseEventMsg{Path: "/tmp/b", Result: shred.Result{FilesShredded: 1}})
	m = updated.(Model)
	for i := 0; i < dissolveFrameCount+1; i++ {
		updated, _ = m.Update(dissolveTickMsg{})
		m = updated.(Model)
	}
	if m.screen != screenDone {
		t.Fatalf("got screen %v, want screenDone after the last target finishes", m.screen)
	}

	updated, _ = m.Update(eraseEventMsg{Done: true, Result: shred.Result{FilesShredded: 2}})
	m = updated.(Model)
	if m.result.FilesShredded != 2 {
		t.Fatalf("got FilesShredded=%d, want 2", m.result.FilesShredded)
	}
}
```

Add `"github.com/brucevanhorn2/erazer/internal/shred"` to `app_test.go`'s imports (needed for `shred.Result` above).

- [ ] **Step 6: Run the tests to verify they fail**

Run: `go test ./internal/ui/...`
Expected: FAIL — `screenErasing`/`eraseEventMsg`/`dissolveTickMsg`/`targetIdx`/`targetDone`/`eventsCh` undefined, and the old `TestConfirmScreen_TriggerErasesSelectedFileAndReachesDone` no longer exists (replaced)

- [ ] **Step 7: Modify internal/ui/app.go**

Add `screenErasing` to the `screen` enum (after `screenConfirm`, before `screenDone`):

```go
const (
	screenBrowsing screen = iota
	screenAbout
	screenConfirm
	screenErasing
	screenDone
)
```

Replace the existing `targets []string` field declaration and the `result shred.Result` / `doneErr string` lines below it with this block (same fields, plus the new ones needed for async erasing):

```go
	targets   []string
	targetIdx int
	opts      shred.Options
	eventsCh  chan shred.Event

	dissolveFrame int
	targetDone    bool

	result  shred.Result
	doneErr string
```

Add `"time"` to the import block (needed by the `tea.Tick` call below).

Replace the `startErase` method entirely:

```go
// startErase validates the Confirm screen's settings, then kicks off an
// async shred of every target: a goroutine runs shred.ShredAll and streams
// one shred.Event per target (plus a final aggregate event) back over
// eventsCh, consumed via the same channel-plus-re-arming-Cmd "subscription"
// pattern exfil/sneakernet use for transfer/backup progress. The Erasing
// screen's own dissolveTick paces the animation independently; each
// target only advances once both its animation has run its course AND its
// real shred.Event has arrived (handleEraseEvent/handleDissolveTick
// below), so the UI can never show a target as erazed before it actually
// is.
func (m Model) startErase() (tea.Model, tea.Cmd) {
	opts, errMsg := m.parseConfirmSettings()
	if errMsg != "" {
		m.confirmErr = errMsg
		return m, nil
	}
	m.opts = opts
	m.targetIdx = 0
	m.dissolveFrame = 0
	m.targetDone = false
	m.result = shred.Result{}
	m.eventsCh = make(chan shred.Event, len(m.targets)+1)
	go shred.ShredAll(m.targets, opts, m.eventsCh)
	m.screen = screenErasing
	return m, tea.Batch(waitForShredEvent(m.eventsCh), dissolveTick())
}

// eraseEventMsg wraps shred.Event as a distinct type so Bubble Tea's
// type-based dispatch in Update routes it correctly.
type eraseEventMsg shred.Event

func waitForShredEvent(ch chan shred.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return eraseEventMsg{Done: true}
		}
		return eraseEventMsg(evt)
	}
}

type dissolveTickMsg struct{}

func dissolveTick() tea.Cmd {
	return tea.Tick(dissolveInterval, func(time.Time) tea.Msg { return dissolveTickMsg{} })
}

func (m Model) handleEraseEvent(msg eraseEventMsg) (tea.Model, tea.Cmd) {
	evt := shred.Event(msg)
	if evt.Done {
		m.result = evt.Result
		return m, nil
	}
	m.targetDone = true
	return m, waitForShredEvent(m.eventsCh)
}

func (m Model) handleDissolveTick() (tea.Model, tea.Cmd) {
	if m.screen != screenErasing {
		return m, nil
	}
	if m.dissolveFrame < dissolveFrameCount {
		m.dissolveFrame++
	}
	if m.dissolveFrame >= dissolveFrameCount && m.targetDone {
		m.targetIdx++
		m.dissolveFrame = 0
		m.targetDone = false
		if m.targetIdx >= len(m.targets) {
			m.screen = screenDone
			return m, nil
		}
	}
	return m, dissolveTick()
}
```

Wire the two new message types into `Update`'s switch (add these cases alongside the existing `tea.WindowSizeMsg`/`tea.KeyMsg` cases):

```go
	case eraseEventMsg:
		return m.handleEraseEvent(msg)

	case dissolveTickMsg:
		return m.handleDissolveTick()
```

- [ ] **Step 8: Modify internal/ui/view.go**

Add the Erasing case to `View`'s switch:

```go
	switch m.screen {
	case screenAbout:
		return m.about.View(m.theme)
	case screenConfirm:
		return m.confirmView()
	case screenErasing:
		return m.erasingView()
	case screenDone:
		return m.doneView()
	default:
		return m.browsingView()
	}
```

Add `"path/filepath"` to the import block, and add `erasingView`:

```go
func (m Model) erasingView() string {
	var b strings.Builder
	b.WriteString(m.theme.Header.Render("Erazing...") + "\n\n")
	for i, t := range m.targets {
		name := filepath.Base(t)
		switch {
		case i < m.targetIdx:
			b.WriteString(m.theme.StatusError.Render("  [erazed] "+name) + "\n")
		case i == m.targetIdx:
			b.WriteString("  " + gradientText(dissolveText(name, m.dissolveFrame, dissolveFrameCount), m.theme.PrimaryColor, m.theme.SecondaryColor) + "\n")
		default:
			b.WriteString("  " + m.theme.BrowserFile.Render(name) + "\n")
		}
	}
	return b.String()
}
```

- [ ] **Step 9: Run the tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/ui
git commit -m "Add Erasing screen: async ShredAll wiring and derez dissolve animation"
```

---

## Task 11: main.go — flag parsing and dispatch

**Files:**
- Modify: `main.go` (replaces the Task 1 stub entirely)
- Test: `main_test.go`

**Interfaces:**
- Consumes: `headless.RunArgs`/`headless.Run` (Task 6), `ui.NewModel` (Task 9).
- Produces: the `erazer` binary's actual CLI surface.

- [ ] **Step 1: Write the failing tests**

`main_test.go`:

```go
package main

import "testing"

func TestParseArgs_NoPathMeansTUI(t *testing.T) {
	got, err := parseArgs([]string{})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got.path != "" {
		t.Fatalf("got path %q, want empty (TUI mode)", got.path)
	}
	if got.passes != 3 {
		t.Fatalf("got passes %d, want default 3", got.passes)
	}
	if got.seed != nil {
		t.Fatal("expected nil seed by default")
	}
}

func TestParseArgs_PathAndFlags(t *testing.T) {
	got, err := parseArgs([]string{"--passes=5", "--seed=42", "-y", "/tmp/secret.csv"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got.path != "/tmp/secret.csv" {
		t.Fatalf("got path %q, want /tmp/secret.csv", got.path)
	}
	if got.passes != 5 {
		t.Fatalf("got passes %d, want 5", got.passes)
	}
	if got.seed == nil || *got.seed != 42 {
		t.Fatalf("got seed %v, want 42", got.seed)
	}
	if !got.assumeYes {
		t.Fatal("expected assumeYes to be true")
	}
}

func TestParseArgs_SeedFlagNotSetStaysNil(t *testing.T) {
	got, err := parseArgs([]string{"/tmp/secret.csv"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if got.seed != nil {
		t.Fatal("expected seed to remain nil when --seed wasn't passed")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test .`
Expected: FAIL — `parseArgs`/`parsedArgs` undefined

- [ ] **Step 3: Write main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/brucevanhorn2/erazer/internal/headless"
	"github.com/brucevanhorn2/erazer/internal/ui"
)

// parsedArgs is main's parsed command-line state, split out from main so
// the parsing logic can be unit tested without touching the global
// flag.CommandLine or actually running the program.
type parsedArgs struct {
	path      string // empty means "launch the TUI"
	passes    int
	seed      *int64
	assumeYes bool
}

func parseArgs(args []string) (parsedArgs, error) {
	fs := flag.NewFlagSet("erazer", flag.ContinueOnError)
	passes := fs.Int("passes", 3, "number of overwrite passes")
	seedFlag := fs.Int64("seed", 0, "deterministic random seed (default: crypto/rand)")
	yes := fs.Bool("yes", false, "skip the confirmation prompt")
	fs.BoolVar(yes, "y", false, "skip the confirmation prompt (shorthand)")
	if err := fs.Parse(args); err != nil {
		return parsedArgs{}, err
	}

	var seed *int64
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			v := *seedFlag
			seed = &v
		}
	})

	var path string
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	return parsedArgs{path: path, passes: *passes, seed: seed, assumeYes: *yes}, nil
}

func main() {
	parsed, err := parseArgs(os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	if parsed.path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		p := tea.NewProgram(ui.NewModel(home), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	code := headless.Run(headless.RunArgs{
		Path:      parsed.path,
		Passes:    parsed.passes,
		Seed:      parsed.seed,
		AssumeYes: parsed.assumeYes,
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
	os.Exit(code)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test .`
Expected: PASS

- [ ] **Step 5: Run the full test suite**

Run: `go build -o erazer . && go vet ./... && go test ./...`
Expected: all PASS, binary builds

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "Wire main.go: flag parsing, headless/TUI dispatch"
```

---

## Task 12: README, and end-to-end manual verification

**Files:**
- Create: `README.md`

**Interfaces:**
- None — documentation and manual verification only.

- [ ] **Step 1: Write README.md**

```markdown
# erazer — secure delete for people who've had a bad week

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A terminal-based secure-delete tool for Linux, built with Go, Bubble Tea, and
Lipgloss — companion to [exfil](https://github.com/brucevanhorn2/exfil) (scp/sftp
client) and [sneakernet](https://github.com/brucevanhorn2/sneakernet) (thumbdrive
backup), in the same cyberpunk house style.

erazer isn't a general system cleaner — it's a focused shredder for the file (or
folder) you need gone for good: a plaintext credential, an old export, anything
`rm` alone doesn't feel like enough for.

**A word on SSDs:** multi-pass overwrite shredding is a real technique on spinning
disks. On flash storage (SSD/NVMe), wear leveling and over-provisioning mean the
logical overwrite may not touch the physical cells that held the original data —
erazer detects this and warns you, but treat it as raising the bar against casual
recovery tools, not a guarantee against determined forensic recovery.

## Status

- ✅ N-pass overwrite + rename-to-garbage + delete, for files and recursively for directories
- ✅ Headless mode: `erazer <path>` validates, confirms, shreds, reports, exits
- ✅ Interactive TUI: browse, multi-select files/folders, confirm with configurable passes/seed, animated erasure
- ✅ Best-effort SSD/NVMe detection with an in-context warning
- ✅ Cyberpunk theming matching exfil/sneakernet's house colors

## Usage

**Headless** — supply a path, it erazes it and exits:

```bash
erazer ~/Downloads/rotated-aws-key.csv
```

Flags:
- `--passes N` — number of overwrite passes (default 3)
- `--seed N` — deterministic random seed for the overwrite data (default: crypto/rand, non-reproducible)
- `-y`, `--yes` — skip the "are you sure" prompt

**Interactive** — no path, launches the TUI to browse to a target:

```bash
erazer
```

## Keybindings

**Browsing**
- `↑`/`k`, `↓`/`j` — move cursor
- `enter`/`l`/`→` — open folder
- `backspace`/`h`/`←` — go up a folder
- `space` — toggle target selection (files and/or folders; folders are shredded recursively)
- `e` — confirm the selected target(s)
- `?` — about screen
- `q` / `ctrl+c` — quit

**Confirm screen**
- `tab` / `shift+tab` — move between the passes field, seed field, and the ERAZE trigger
- typing — edit the focused field
- `enter` — advance focus, or trigger the erase when ERAZE is focused
- `esc` — back to browsing

**Done screen**
- any key — quit

## Building

```bash
go build -o erazer .
```

## Architecture

```
main.go                     entrypoint — dispatches headless vs TUI
internal/shred/
  engine.go                  Shred(path, opts) — recursive overwrite + delete
  overwrite.go                 N-pass overwrite, rename-to-garbage, unlink
  rotational.go                 best-effort SSD/NVMe detection
  progress.go                    ShredAll — multi-target streaming for the UI/headless path
internal/browse/
  selection.go                 inherited-override selection tree (same design as sneakernet's)
internal/headless/
  run.go                        CLI path: validate, confirm, shred, report, exit code
internal/ui/
  theme.go / gradient.go         cyberpunk theme and gradient chrome
  about.go                        ASCII logo, about screen
  browser.go                      single-pane file browser
  app.go / view.go / dissolve.go   screen state machine (Browsing → Confirm → Erasing → Done)
```

## License

MIT — see [LICENSE](LICENSE).
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "Add README"
```

- [ ] **Step 3: Manual end-to-end verification**

Run:

```bash
cd /home/bruce/Projects/erazer
go build -o erazer .
mkdir -p /tmp/erazer-smoke && echo "AKIA-FAKE-KEY-FOR-SMOKE-TEST" > /tmp/erazer-smoke/fake-key.txt
./erazer --yes /tmp/erazer-smoke/fake-key.txt
ls /tmp/erazer-smoke
```

Expected: the `erazer: erazed ...` / summary output prints, and `ls /tmp/erazer-smoke` shows the directory is empty.

Then run `./erazer` with no arguments, and manually confirm in the terminal:
- The browser shows your home directory and responds to arrow keys.
- `space` on a throwaway test file marks it (☑), `e` opens the Confirm screen with that path listed.
- Typing in the passes/seed fields works, `tab` cycles focus, the ERAZE trigger renders in red and more strongly highlighted when focused.
- Pressing `enter` on the trigger shows the Erasing screen with the filename visibly glitching before the screen advances to Done with a correct summary.
- The target file is actually gone from disk afterward.
- `?` from Browsing shows the About screen with the erazer logo and tagline; any key returns to Browsing.

Report back any visual or behavioral issues found during this manual pass before considering the plan complete.
