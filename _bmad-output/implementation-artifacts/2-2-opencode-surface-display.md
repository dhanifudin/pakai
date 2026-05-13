# Story 2.2: OpenCode Surface Display

Status: review

## Story

As a terminal user,
I want `pakai tmux`, `pakai waybar`, and `pakai status` to show OpenCode usage alongside Claude,
so that I can see my total AI spending across providers at a glance.

## Acceptance Criteria

1. **Given** no manual limit is configured for OpenCode (`provider.opencode.limit` is unset), **When** I run `pakai tmux`, **Then** it prints both providers (e.g. `claude:72% | opencode:$1.24`) — OpenCode shows absolute USD cost, not a percentage

2. **Given** a manual limit is configured (e.g. `provider.opencode.limit = 10.00`), **When** I run `pakai tmux`, **Then** OpenCode displays as a percentage of the configured limit (e.g. `opencode:12%`)

3. **Given** OpenCode usage exceeds the configured limit (e.g. $11.20 against a $10.00 limit), **When** I run `pakai tmux`, **Then** it shows `opencode:112%` — not capped at 100%

4. **Given** OpenCode has `state: "error"`, **When** I run `pakai tmux`, **Then** it prints `opencode:??` — Claude's entry is unaffected and shown normally

5. **Given** both providers have `state: "ok"`, **When** I run `pakai waybar`, **Then** the JSON `text` field includes both providers and `class` reflects the worst state across all providers (e.g. if one is `ok` and one is `over-limit`, `class` is `"over-limit"`)

6. **Given** both providers are present, **When** I run `pakai status`, **Then** the human-readable summary shows a row for each provider with name, current usage value, and state

7. **Given** OpenCode has `state: "ok"` with no configured limit, **When** I run `pakai waybar`, **Then** the `class` field is `"ok"` and `tooltip` shows the absolute cost (e.g. `OpenCode: $1.24 this month`)

## Tasks / Subtasks

- [x] Update `internal/renderer/tmux.go` for multi-provider + dual display modes (AC: 1, 2, 3, 4)
  - [x] If `Usage.Cost != nil && Usage.Percent == nil`: format as `provider:$X.XX` (absolute USD)
  - [x] If `Usage.Percent != nil`: format as `provider:X%` (percentage — includes over-limit case)
  - [x] If `Usage.State == StateError || StateStale`: format as `provider:??`
  - [x] Join multiple providers with separator (default: ` | `)
- [x] Update `internal/renderer/waybar.go` for multi-provider worst-state class (AC: 5, 7)
  - [x] `class` = worst state across all providers (precedence: `over-limit` > `error` > `stale` > `ok`)
  - [x] `tooltip` for OpenCode without limit: `"OpenCode: $X.XX this month"`
  - [x] `text` field includes both providers
- [x] Update `internal/renderer/status.go` for multi-provider rows (AC: 6)
  - [x] One row per provider: name, value, state
- [x] Add `provider.opencode.limit` to config schema (AC: 2, 3)
  - [x] Optional float field in config TOML
  - [x] When set: compute `percent = (cost / limit) * 100`, populate `Usage.Percent`
  - [x] When unset: `Usage.Percent` remains nil; `Usage.Cost` is set
- [x] Update golden test files for multi-provider scenarios (AC: 1, 4, 5, 7)
  - [x] Add golden files: `tmux_two_providers_ok.golden`, `tmux_opencode_error.golden`
  - [x] `waybar_two_ok.golden`, `waybar_worst_state.golden`

## Dev Notes

- **Prerequisite:** Story 2.1 (OpenCode adapter) must be complete. This story extends the renderers to handle the second provider.
- **Dual display mode logic:** The renderer determines display format from the `Usage` struct fields:
  - `Cost != nil && Percent == nil` → absolute USD: `$X.XX` (FR5)
  - `Percent != nil` → percentage: `X%` — raw value, never capped (FR6, FR7)
  - `State == StateError || StateStale` → `??` (FR47)
  - When `provider.opencode.limit` is configured in config, the adapter or poll loop must compute and populate `Usage.Percent` from `cost/limit*100` before storing to cache
- **Where to compute percent from cost+limit:** The poll loop (in `daemon/`) reads `config.Config()` fresh each cycle. After the OpenCode adapter returns `Usage{Cost: &cost}`, the poll loop checks `config.Config().Provider.OpenCode.Limit`; if set, it computes and fills `Usage.Percent`. This keeps the adapter pure and the config live-reload contract intact.
- **Worst-state class logic for waybar (AC 5):**
  ```go
  // Precedence: over-limit > error > stale > ok
  func worstClass(usages []model.Usage) string { ... }
  ```
  This covers the 4 states from Epic 1. `warning` and `critical` are added in Epic 4 (Story 4.3). The precedence order there will be: `over-limit > critical > warning > ok > stale > error`.
- **USD formatting:** Use `fmt.Sprintf("$%.2f", cost)` — two decimal places, dollar sign prefix.
- **Separator between providers:** Default ` | `. This will be made configurable in Epic 3, Story 3.4. For now, hardcode the default.
- **Provider isolation in waybar class:** If one provider is `error` and another is `ok`, `class = "error"`. Each provider's state is independent but the composite `class` reflects the worst.

### Project Structure Notes

Files to UPDATE (not create new):
```
internal/renderer/tmux.go      ← extend for dual display mode + multi-provider
internal/renderer/waybar.go    ← extend for worst-state class, multi-provider tooltip
internal/renderer/status.go    ← extend for multi-provider rows
internal/renderer/tmux_test.go ← add multi-provider golden cases
internal/renderer/waybar_test.go ← add worst-state golden cases
```

New testdata golden files for multi-provider scenarios.

Config schema change:
```toml
# config.toml
[provider.opencode]
limit = 10.00   # optional; when set, display as %
```

### References

- [Source: architecture.md#Data Architecture] — Usage struct: Percent *float64, Cost *float64
- [Source: architecture.md#Format Patterns] — waybar JSON shape, class field values
- [Source: architecture.md#Process Patterns] — stale vs error, provider isolation
- [Source: project_pakai_architecture.md#OpenCode Go Adapter] — no subscription limits stored locally; user sets manual limit via config
- [Source: epics.md#Story 2.2] — acceptance criteria

## Dev Agent Record

### Agent Model Used

deepseek-v4-pro

### Debug Log References

### Completion Notes List

- Updated `internal/renderer/waybar.go` — added proper class precedence (`over-limit` > `error` > `stale` > `ok`) using `worseClass()` function and `classPrecedence` map; changed no-limit tooltip from "(no limit)" to "this month"
- Added multi-provider golden tests: `tmux_two_ok`, `tmux_opencode_error`, `waybar_two_ok`, `waybar_worst_state`
- tmux and status renderers already handled multi-provider display correctly from story 1.4
- Config already supports `provider.opencode.limit` — accessor reads from live config

### File List

- `internal/renderer/waybar.go` (modified)
- `internal/renderer/tmux_test.go` (modified)
- `internal/renderer/waybar_test.go` (modified)
- `internal/renderer/testdata/tmux_two_ok.golden` (new)
- `internal/renderer/testdata/tmux_opencode_error.golden` (new)
- `internal/renderer/testdata/waybar_two_ok.golden` (new)
- `internal/renderer/testdata/waybar_worst_state.golden` (new)

## Change Log

- 2026-05-12: Story 2.2 implementation — worst-state class precedence, "this month" tooltip, multi-provider golden tests
