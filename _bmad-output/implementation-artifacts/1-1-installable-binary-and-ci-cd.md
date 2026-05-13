# Story 1.1: Installable Binary & CI/CD

Status: review

## Story

As a developer,
I want to install pakai with `go install github.com/dhanifudin/pakai/cmd/pakai@latest` and run `pakai version`,
so that I can verify the tool is installed and working before any providers are configured.

## Acceptance Criteria

1. **Given** the Go toolchain is installed with no C compiler, **When** I run `go install github.com/dhanifudin/pakai/cmd/pakai@latest`, **Then** the binary builds successfully with `CGO_ENABLED=0` and is placed in `$GOPATH/bin/pakai`

2. **Given** pakai is installed, **When** I run `pakai version`, **Then** it prints the version string and exits with code 0

3. **Given** pakai is installed, **When** I run `pakai --help`, **Then** it lists available subcommands with descriptions and exits with code 0

4. **Given** any pakai subcommand encounters a fatal error, **When** the command exits, **Then** it exits with a non-zero status code and prints the error to stderr

5. **Given** a PR is opened or a push lands on main, **When** GitHub Actions runs the CI workflow, **Then** `go test ./...`, `go vet ./...`, and `staticcheck` all pass on Linux/amd64

6. **Given** a tag matching `v*` is pushed, **When** goreleaser runs the release workflow, **Then** Linux/amd64 and Linux/arm64 binaries are produced and attached to a GitHub release with `CGO_ENABLED=0`

## Tasks / Subtasks

- [x] Initialize Go module (AC: 1)
  - [x] Run `go mod init github.com/dhanifudin/pakai`
  - [x] Add all required dependencies: cobra, BurntSushi/toml, modernc.org/sqlite, golang.org/x/sync, fsnotify/fsnotify, stretchr/testify
- [x] Create `internal/model/model.go` with `Usage`, `State` types (AC: 1)
  - [x] Define `State` type with `StateOK`, `StateError`, `StateStale` constants
  - [x] Define `Usage` struct with `Provider`, `Label`, `State`, `Percent *float64`, `Cost *float64`, `RefreshedAt time.Time`, `ErrorMsg string`
- [x] Create `cmd/pakai/main.go` with cobra root command (AC: 2, 3, 4)
  - [x] Register `version` subcommand (prints version string, exit 0)
  - [x] Ensure all fatal errors print to stderr and exit non-zero
- [x] Create `.github/workflows/ci.yml` (AC: 5)
  - [x] Trigger on PR and push to main
  - [x] Steps: `go test ./...`, `go vet ./...`, `staticcheck`
  - [x] Run on Linux/amd64
- [x] Create `.github/workflows/release.yml` and `.goreleaser.yaml` (AC: 6)
  - [x] Trigger on `v*` tag push
  - [x] Build Linux/amd64 and Linux/arm64 with `CGO_ENABLED=0`
  - [x] Attach binaries to GitHub release
- [x] Create `Makefile` with `build`, `test`, `lint`, `run-daemon` targets
- [x] Create `.gitignore`

## Dev Notes

- **This is story 1.1 — greenfield initialization.** No existing code to read. Create the entire project skeleton.
- **`internal/model/` MUST be implemented here** — it is the dependency root. All other packages import it; it imports nothing. Every subsequent story depends on these types being correct.
- **Dependency list (exact):** `github.com/spf13/cobra`, `github.com/BurntSushi/toml`, `modernc.org/sqlite`, `golang.org/x/sync` (import path: `golang.org/x/sync/singleflight`), `github.com/fsnotify/fsnotify`, `github.com/stretchr/testify` (test only). Do NOT add viper — it silently normalizes config keys to lowercase, corrupting provider-namespaced keys.
- **CGO_ENABLED=0 is mandatory** — the modernc.org/sqlite driver is pure Go. The goreleaser config must set this. CI must verify CGo-free builds.
- **Error output goes to stderr** — use `cmd.SetErr(os.Stderr)` on the cobra root; all `fmt.Fprintf` error output targets `os.Stderr`.
- **Logging:** `log/slog` only across all packages from the start. Zero `fmt.Println` or `log.Printf` in non-test code.
- **Go version:** 1.21+ required for `log/slog` stdlib inclusion.

### Project Structure Notes

```
github.com/dhanifudin/pakai/
├── .github/workflows/ci.yml
├── .github/workflows/release.yml
├── .goreleaser.yaml
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
├── cmd/pakai/main.go       ← cobra root + version subcommand
└── internal/
    └── model/
        └── model.go        ← Usage, State — imports nothing
```

Only create packages needed for this story. Do not stub out future packages.

### References

- [Source: architecture.md#Package Layout] — full directory tree
- [Source: architecture.md#Initialization] — exact dependency list and go get commands
- [Source: architecture.md#Naming Patterns] — PascalCase exports, camelCase unexported, snake_case files
- [Source: architecture.md#Infrastructure & Deployment] — CI/CD pipeline and goreleaser spec
- [Source: epics.md#Story 1.1] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Created `internal/model/model.go` with `State` type (`StateOK`, `StateError`, `StateStale`) and `Usage` struct (`Provider`, `Label`, `State`, `Percent *float64`, `Cost *float64`, `RefreshedAt time.Time`, `ErrorMsg`)
- Added `version` subcommand to `cmd/pakai/main.go` using ldflags-injectable `version` variable (`-X main.version=...`)
- Set `rootCmd.SetErr(os.Stderr)` for proper error routing
- Created `.github/workflows/ci.yml` — triggers on PR/push to main, runs `go vet`, `staticcheck`, `go test` on Linux/amd64
- Created `.github/workflows/release.yml` + `.goreleaser.yaml` — `v*` tag triggers goreleaser, builds Linux/amd64+arm64 with `CGO_ENABLED=0`
- Created `Makefile` with `build`, `test`, `lint`, `run-daemon` targets
- Created `.gitignore`
- Added missing dependencies: `BurntSushi/toml`, `golang.org/x/sync`, `stretchr/testify`
- Tests added: `internal/model/model_test.go` (4 tests), `cmd/pakai/main_test.go` (3 tests)
- All ACs verified: binary builds with CGO_ENABLED=0, `pakai version` works, `--help` lists subcommands, fatal errors exit non-zero

### File List

- `internal/model/model.go` (new)
- `internal/model/model_test.go` (new)
- `cmd/pakai/main.go` (modified)
- `cmd/pakai/main_test.go` (new)
- `.github/workflows/ci.yml` (new)
- `.github/workflows/release.yml` (new)
- `.goreleaser.yaml` (new)
- `Makefile` (new)
- `.gitignore` (new)
- `go.mod` (modified)
- `go.sum` (modified)

## Change Log

- 2026-05-12: Story 1.1 implementation — project skeleton, model package, version command, CI/CD, Makefile, .gitignore
