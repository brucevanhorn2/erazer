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
