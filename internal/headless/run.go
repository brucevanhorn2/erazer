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
