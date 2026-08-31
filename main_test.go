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
