# Story 3.5: Shell Completions & Setup Live Preview

Status: review

## Tasks / Subtasks

- [x] Add `pakai completion` subcommand (AC: 1, 2, 3, 4)
- [x] Add live preview step to `pakai setup` (AC: 5, 6)

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Completion Notes List

- Added `newCompletionCmd()` — cobra's `GenBashCompletion`, `GenZshCompletion`, `GenFishCompletion` for bash/zsh/fish
- Added `runLivePreview()` — auto-spawns daemon, fetches /status, renders tmux + waybar output; errors print to stderr but don't fail setup (AC 6)
- Live preview shows final output section in setup: `--- Live Preview ---` with both tmux and waybar renderings

### File List
- `cmd/pakai/main.go` (modified)

## Change Log
- 2026-05-12: Story 3.5 — shell completions (bash/zsh/fish), setup live preview
