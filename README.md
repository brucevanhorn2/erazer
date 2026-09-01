# erazer — secure delete for people who've had a bad week

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A terminal-based secure-delete tool for Linux, built with Go, Bubble Tea, and
Lipgloss — companion to [exfil](https://github.com/brucevanhorn2/exfil) (scp/sftp
client) and [sneakernet](https://github.com/brucevanhorn2/sneakernet) (thumbdrive
backup), in the same cyberpunk house style.

erazer isn't a general system cleaner — it's a focused shredder for the file (or
folder) you need gone for good: a plaintext credential, an old export, anything
`rm` alone doesn't feel like enough for.

**A word on SSDs:** multi-pass overwrite shredding is a real technique on spinning
disks. On flash storage (SSD/NVMe), wear leveling and over-provisioning mean the
logical overwrite may not touch the physical cells that held the original data —
erazer detects this and warns you, but treat it as raising the bar against casual
recovery tools, not a guarantee against determined forensic recovery.

## Status

- ✅ N-pass overwrite + rename-to-garbage + delete, for files and recursively for directories
- ✅ Headless mode: `erazer <path>` validates, confirms, shreds, reports, exits
- ✅ Interactive TUI: browse, multi-select files/folders, confirm with configurable passes/seed, animated erasure
- ✅ Best-effort SSD/NVMe detection with an in-context warning
- ✅ Cyberpunk theming matching exfil/sneakernet's house colors

## Usage

**Headless** — supply a path, it erazes it and exits:

```bash
erazer ~/Downloads/rotated-aws-key.csv
```

Flags:
- `--passes N` — number of overwrite passes (default 3)
- `--seed N` — deterministic random seed for the overwrite data (default: crypto/rand, non-reproducible)
- `-y`, `--yes` — skip the "are you sure" prompt

**Interactive** — no path, launches the TUI to browse to a target:

```bash
erazer
```

## Keybindings

**Browsing**
- `↑`/`k`, `↓`/`j` — move cursor
- `enter`/`l`/`→` — open folder
- `backspace`/`h`/`←` — go up a folder
- `space` — toggle target selection (files and/or folders; folders are shredded recursively)
- `e` — confirm the selected target(s)
- `?` — about screen
- `q` / `ctrl+c` — quit

**Confirm screen**
- `tab` / `shift+tab` — move between the passes field, seed field, and the ERAZE trigger
- typing — edit the focused field
- `enter` — advance focus, or trigger the erase when ERAZE is focused
- `esc` — back to browsing

**Done screen**
- any key — quit

## Building

```bash
go build -o erazer .
```

## Architecture

```
main.go                     entrypoint — dispatches headless vs TUI
internal/shred/
  engine.go                  Shred(path, opts) — recursive overwrite + delete
  overwrite.go                 N-pass overwrite, rename-to-garbage, unlink
  rotational.go                 best-effort SSD/NVMe detection
  progress.go                    ShredAll — multi-target streaming for the UI/headless path
internal/browse/
  selection.go                 inherited-override selection tree (same design as sneakernet's)
internal/headless/
  run.go                        CLI path: validate, confirm, shred, report, exit code
internal/ui/
  theme.go / gradient.go         cyberpunk theme and gradient chrome
  about.go                        ASCII logo, about screen
  browser.go                      single-pane file browser
  app.go / view.go / dissolve.go   screen state machine (Browsing → Confirm → Erasing → Done)
```

## License

MIT — see [LICENSE](LICENSE).
