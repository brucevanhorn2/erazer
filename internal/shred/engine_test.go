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

func TestShred_SkipFunctionPreservesExcludedDescendant(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.txt")
	gone := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(keep, []byte("keep me"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(gone, []byte("erase me"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	res := Shred(dir, Options{Passes: 1, Skip: func(p string) bool { return p == keep }})
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("expected kept file to survive, got err=%v", err)
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Fatalf("expected other file to be gone, got err=%v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected the directory itself to survive since it still has a kept child, got err=%v", err)
	}
}

// TestShred_SkipInNestedSubdirectoryDoesNotReportSpuriousParentErrors
// guards against a narrower (and wrong) fix for the ENOTEMPTY-swallow
// scope: one that only tracked *direct* skipped children of a directory
// would still misfire here, because the outer target directory doesn't
// skip anything itself — the skip only fires two levels down. Its own
// os.Remove will still return ENOTEMPTY (because "mid" survives), and
// that must be recognized as an accounted-for outcome, not reported as a
// FileError.
func TestShred_SkipInNestedSubdirectoryDoesNotReportSpuriousParentErrors(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "outer")
	mid := filepath.Join(target, "mid")
	if err := os.MkdirAll(mid, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	keep := filepath.Join(mid, "keep.txt")
	gone := filepath.Join(target, "gone.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(gone, []byte("gone"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	res := Shred(target, Options{Passes: 1, Skip: func(p string) bool { return p == keep }})
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("expected the deeply-kept file to survive, got err=%v", err)
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Fatalf("expected the sibling file to be gone, got err=%v", err)
	}
	if _, err := os.Stat(mid); err != nil {
		t.Fatalf("expected mid (ancestor of the kept file) to survive, got err=%v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target to survive since it transitively holds a kept file, got err=%v", err)
	}
}

// TestRemoveIfExists_UnexpectedENOTEMPTYIsReported is the complement of
// the skip tests above: when a directory can't be removed for a reason
// the caller did NOT already account for (expectNonEmpty=false) — e.g. an
// unrelated process wrote into it during the run — that must still surface
// as a FileError rather than being silently swallowed.
func TestRemoveIfExists_UnexpectedENOTEMPTYIsReported(t *testing.T) {
	dir := t.TempDir()
	nonEmpty := filepath.Join(dir, "surprise")
	if err := os.Mkdir(nonEmpty, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmpty, "unexpected.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var res Result
	removed := removeIfExists(nonEmpty, &res, false)
	if removed {
		t.Fatal("expected removeIfExists to report failure for a non-empty directory")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected the unexpected ENOTEMPTY to be reported as an error, got %v", res.Errors)
	}
	if _, err := os.Stat(nonEmpty); err != nil {
		t.Fatalf("expected the directory to survive, got err=%v", err)
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
