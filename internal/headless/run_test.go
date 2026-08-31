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

func TestRun_ShredFailure_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.txt")
	if err := os.WriteFile(path, []byte("data"), 0444); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	var stdout, stderr bytes.Buffer
	code := Run(RunArgs{
		Path: path, Passes: 1, AssumeYes: true,
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("got exit code %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.String() == "" {
		t.Fatal("expected the shred error to be reported on stderr")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected the unshreddable file to survive, got err=%v", err)
	}
}
