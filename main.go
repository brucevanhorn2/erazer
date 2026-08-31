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
