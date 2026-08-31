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
