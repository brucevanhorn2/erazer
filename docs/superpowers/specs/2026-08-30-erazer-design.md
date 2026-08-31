# erazer — Design

## Purpose

A secure-delete tool for Linux, in the same house style as [exfil](../../../exfil) (scp/sftp
TUI) and [sneakernet](../../../sneakernet) (thumbdrive backup TUI): Go + Bubbletea +
Lipgloss, cyberpunk theming, single lowercase-word naming.

Motivating use case: after rotating a leaked AWS key, the plaintext key (and a CSV export
of it) needs to be destroyed more thoroughly than a normal `rm`. erazer is a focused
secure-delete/shredder tool — it is not a full BleachBit-style system cleaner (browser
cache, package manager cache, trash, etc. are explicitly out of scope).

## Two entry modes

1. **`erazer <path>`** — headless. Validate the path exists, prompt `y/n` unless
   `-y`/`--yes` is passed, run the shred, print per-file progress lines to stdout, exit
   non-zero if any target failed.
2. **`erazer`** (no args) — launches the Bubbletea TUI: browse to one or more targets,
   confirm/configure on a settings panel, watch the erase animate, see a summary.

## Package layout

Flat layout (matches sneakernet, not exfil's nested `cmd/`):

```
main.go                  entrypoint — dispatches headless vs TUI based on os.Args
internal/shred/
  engine.go              Shred(path, opts) — walks path, shreds each file, removes empty dirs
  overwrite.go           N-pass overwrite (crypto/rand, or seeded math/rand), truncate,
                         rename-to-garbage, unlink
  rotational.go          best-effort SSD/NVMe detection via /sys/block rotational flag
internal/browse/
  selection.go           adapted from sneakernet's inherited-override selection tree
                         (multi-target picking)
internal/ui/
  theme.go               house colors — primary #B341F5 / secondary #6E6E6E, cyan/green/red
                         accents (same values as exfil/sneakernet)
  gradient.go            gradient border/text helper (same technique as exfil/sneakernet)
  about.go               toilet-font ASCII logo, cyan→purple gradient, about screen
  browser.go             single-pane file browser
  settings.go            right-hand panel: pass count, seed, big red "ERAZE" trigger
  derez.go               campy wipe animation — filename glitches/dissolves char-by-char
                         during shredding
  app.go                 screen state machine: Browsing → Confirm → Erazing → Done
  view.go                per-screen rendering
internal/headless/
  run.go                 CLI path: validate target, confirm unless -y, invoke shred engine,
                         set exit code
```

Module path: `github.com/brucevanhorn2/erazer` (matches the actual GitHub remote —
exfil's `go.mod` uses a legacy mismatched path; sneakernet's is correct and is the pattern
to follow).

## Shred engine

- `shred.Options{Passes int, Seed *int64}`. Default `Passes` is 3. `Seed == nil` means
  each pass uses `crypto/rand`; a set seed switches to deterministic `math/rand` for that
  run — this satisfies the user-facing "random seed" setting and doubles as the mechanism
  for deterministic unit tests.
- Per regular file: for each pass, seek to 0, overwrite the full file length with random
  bytes, `Sync()`. After all passes: truncate to 0, rename to a random garbage name in the
  same directory, then remove. This scrubs both content and filename.
- Directory target: `filepath.WalkDir`, shred every regular file found; symlinks are
  removed without being followed (never overwrite through a symlink); once a directory's
  contents are gone, remove it too, walking bottom-up so parents are empty when reached.
- Non-regular, non-symlink files (devices, sockets, FIFOs) are just unlinked — nothing to
  overwrite.

## Rotational (SSD) detection

Best-effort: resolve the target's backing mount via `/proc/mounts`, then read
`/sys/block/<dev>/queue/rotational`. A `0` (non-rotational) surfaces a caveat:

- TUI: a warning banner on the Confirm/Settings screen.
- Headless: a note printed to stderr before the confirmation prompt.

Wording should be accurate, not alarmist: overwrite-based shredding is a real technique on
spinning disks, but on flash media, wear leveling and over-provisioning mean the logical
overwrite may not touch the physical cells that held the original data — so this raises
the bar against casual undelete/recovery tools, but is not a guarantee against determined
forensic recovery the way full-disk encryption is.

If detection fails for any reason (unusual mount, container, permission denied reading
sysfs), skip the warning silently rather than blocking the operation — this is a UX nicety,
not a safety gate.

## TUI flow

1. **Browsing** — single-pane file browser (adapted from sneakernet's `browser.go`).
   `enter`/`l`/`→` opens a directory (navigation, same as sneakernet), `space` toggles a
   target's selection; supports selecting multiple files and/or directories in one run,
   reusing the same inherited-override selection tree sneakernet uses for its
   backup-selection UI. `e` (mirrors sneakernet's `b` for "start backup") advances to
   Confirm once at least one target is selected.
2. **Confirm/Settings** — right-hand panel: lists selected target(s), a pass-count field
   (default 3), an optional seed field (blank = crypto/rand), the SSD caveat banner when
   applicable, and a bold red "ERAZE" trigger. This screen is the single confirmation gate
   — no separate y/n prompt stacked on top of it.
3. **Erazing** — progress bar per file, plus the derez effect: each filename visually
   glitches/dissolves character-by-character through the theme's gradient colors before
   vanishing from the list. Built with plain lipgloss/string tricks (character
   substitution + color cycling on a timer via a re-arming `tea.Cmd`, the same
   "subscription" pattern exfil uses for transfer progress) — no animation library.
4. **Done** — summary (files shredded, total bytes overwritten, any errors), any key to
   quit.

## Error handling

Per-file errors (permission denied, file raced out from under us) are reported and
skipped rather than aborting the whole batch — matches exfil's existing philosophy for
multi-file operations (one bad subtree doesn't kill the operation). A target that's
already gone by the time it's processed counts as success (the goal — the data not
existing — is already achieved). Headless mode's exit code is non-zero if any target
failed.

## Testing

- `internal/shred`: unit tests using a fixed seed — verify each pass's content actually
  changes the file (not a no-op), verify final rename+delete happens and the original
  filename no longer exists, verify directory recursion reaches nested files and symlinks
  aren't followed.
- `internal/browse`: adapt sneakernet's existing selection tests.
- No TUI screenshot testing — matches the existing convention in both exfil and
  sneakernet.

## Repo deliverables

- `README.md` — house-style tagline, badges, screenshot/about-screen image placeholder,
  usage, keybindings, architecture summary — same shape as exfil's and sneakernet's
  READMEs.
- `LICENSE` — MIT, matching exfil and sneakernet.

## Out of scope (v1)

- BleachBit-style cache/junk cleaning categories (browser cache, package manager cache,
  trash, logs). erazer is a shredder, not a system cleaner.
- Whole-disk / free-space wiping.
- Settings persistence, lingo packs, or a Settings screen beyond the per-run
  pass-count/seed fields on the Confirm screen.
- Windows/macOS support — Linux only, consistent with exfil/sneakernet.
