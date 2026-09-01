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
