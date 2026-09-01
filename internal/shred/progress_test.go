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
