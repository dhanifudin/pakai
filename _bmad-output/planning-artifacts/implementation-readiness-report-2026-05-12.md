---
stepsCompleted: [1, 2, 3, 4, 5, 6]
status: 'complete'
completedAt: '2026-05-12'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
  - '_bmad-output/planning-artifacts/epics.md'
---

# Implementation Readiness Assessment Report

**Date:** 2026-05-12
**Project:** pakai

## PRD Analysis

### Functional Requirements

50 FRs extracted across 7 categories:
- Provider Data Reading: FR1–FR9
- Daemon Management: FR10–FR17
- Status Surface Delivery: FR18–FR26
- Setup & Configuration: FR27–FR35
- Alert & Adaptive Refresh: FR36–FR39
- Developer & Integration Support: FR40–FR45
- Graceful Degradation: FR46–FR50

### Non-Functional Requirements

23 NFRs extracted across 6 categories:
- Performance: NFR1–NFR5 (200ms surface response, 50ms HTTP, 10 SSE clients, goroutine separation)
- Reliability: NFR6–NFR10 (7+ day uptime, systemd restart, no goroutine leaks, no stale PID)
- Security: NFR11–NFR14 (127.0.0.1 only, no telemetry, read-only access, no privilege escalation)
- Portability: NFR15–NFR18 (CGo-free, Linux/amd64+arm64, XDG paths, go install)
- Observability: NFR19–NFR22 (parse errors surfaced, stderr logging, /health fields, provider debug)
- Accessibility: NFR23 (dual signal: CSS class + numeric value)

### PRD Completeness Assessment

PRD is complete and well-structured. All requirements are numbered, testable, and unambiguous. No conflicts or contradictions detected between FRs. NFRs have specific measurable targets (200ms, 50ms, 7+ days, 10 clients).

## Epic Coverage Validation

### Coverage Matrix

| FR | PRD Requirement (summary) | Epic → Story | Status |
|---|---|---|---|
| FR1 | Read Claude usage from local stats cache file | Epic 1 → Story 1.3 | ✅ Covered |
| FR2 | Query OpenCode usage from local SQLite DB | Epic 2 → Story 2.1 | ✅ Covered |
| FR3 | Aggregate OpenCode usage by provider/billing month | Epic 2 → Story 2.1 | ✅ Covered |
| FR4 | Express Claude usage as % of subscription limit | Epic 1 → Story 1.3 | ✅ Covered |
| FR5 | Express OpenCode as absolute USD when no limit | Epic 2 → Story 2.2 | ✅ Covered |
| FR6 | Express OpenCode as % when manual limit configured | Epic 2 → Story 2.2 | ✅ Covered |
| FR7 | Display over-limit usage as-is (e.g. 105%) | Epic 1 → Story 1.3 | ✅ Covered |
| FR8 | Distinguish inaccessible vs stale data — named states | Epic 1 → Story 1.3 | ✅ Covered |
| FR9 | Indicate when provider data was last refreshed | Epic 1 → Story 1.3 | ✅ Covered |
| FR10 | Serve current usage data on local HTTP endpoint | Epic 1 → Story 1.3 | ✅ Covered |
| FR11 | Stream live updates via SSE | Epic 1 → Story 1.5 | ✅ Covered |
| FR12 | Report health, uptime, connections, port | Epic 1 → Story 1.2 | ✅ Covered |
| FR13 | Auto-start daemon on CLI subcommand if port unreachable | Epic 1 → Story 1.4 | ✅ Covered |
| FR14 | Detect port conflict and report explicitly | Epic 1 → Story 1.2 | ✅ Covered |
| FR15 | Explicit daemon start/stop/status CLI commands | Epic 1 → Story 1.2 | ✅ Covered |
| FR16 | Orderly shutdown on termination signal | Epic 1 → Story 1.2 | ✅ Covered |
| FR17 | New SSE subscriber receives current state immediately | Epic 1 → Story 1.5 | ✅ Covered |
| FR18 | Compact plain-text output for tmux | Epic 1 → Story 1.4 | ✅ Covered |
| FR19 | Structured JSON payload for waybar | Epic 1 → Story 1.4 | ✅ Covered |
| FR20 | Human-readable multi-provider summary | Epic 1 → Story 1.4 | ✅ Covered |
| FR21 | Configurable per-provider display labels | Epic 3 → Story 3.4 | ✅ Covered |
| FR22 | Configurable separator character | Epic 3 → Story 3.4 | ✅ Covered |
| FR23 | CSS class field with ok/warning/critical/over-limit/error/stale | Epic 1 → Story 1.4 (partial) + Epic 4 → Story 4.3 | ✅ Covered |
| FR24 | `pakai status` machine-readable JSON mode | Epic 1 → Story 1.4 | ✅ Covered |
| FR25 | Configurable display order of providers | Epic 3 → Story 3.4 | ✅ Covered |
| FR26 | Live preview of surface output after setup | Epic 3 → Story 3.5 | ✅ Covered |
| FR27 | Setup detects installed providers by file path | Epic 3 → Story 3.2 | ✅ Covered |
| FR28 | Setup creates systemd user service unit | Epic 3 → Story 3.2 | ✅ Covered |
| FR29 | Setup prints copy-pasteable surface config snippet | Epic 3 → Story 3.2 | ✅ Covered |
| FR30 | Re-running setup is idempotent, preserves existing config | Epic 3 → Story 3.2 | ✅ Covered |
| FR31 | Read/write/list individual config values from CLI | Epic 3 → Story 3.3 | ✅ Covered |
| FR32 | Config changes take effect within one poll cycle | Epic 3 → Story 3.1 | ✅ Covered |
| FR33 | Individual providers can be enabled/disabled | Epic 3 → Story 3.3 | ✅ Covered |
| FR34 | Default config when no file present | Epic 3 → Story 3.1 | ✅ Covered |
| FR35 | Config validation on write, reject invalid values | Epic 3 → Story 3.3 | ✅ Covered |
| FR36 | Configurable percentage thresholds for alert zones | Epic 4 → Story 4.1 | ✅ Covered |
| FR37 | Daemon auto-adjusts poll interval by usage/threshold | Epic 4 → Story 4.2 | ✅ Covered |
| FR38 | Surface outputs reflect current threshold zone via CSS class | Epic 4 → Story 4.3 | ✅ Covered |
| FR39 | Threshold config applies uniformly across all providers | Epic 4 → Story 4.1 | ✅ Covered |
| FR40 | Inject synthetic usage data; held in memory only | Epic 5 → Story 5.1 | ✅ Covered |
| FR41 | Remove injected data to restore live readings | Epic 5 → Story 5.1 | ✅ Covered |
| FR42 | Third-party tools can query /status | Epic 1 → Story 1.4 | ✅ Covered |
| FR43 | Third-party tools can subscribe to SSE /events | Epic 1 → Story 1.5 | ✅ Covered |
| FR44 | Shell completion scripts for bash, zsh, fish | Epic 3 → Story 3.5 | ✅ Covered |
| FR45 | Exit code 0 on success, non-zero on error | Epic 1 → Story 1.1 | ✅ Covered |
| FR46 | Per-provider independent health state | Epic 1 → Story 1.3 | ✅ Covered |
| FR47 | Explicit unavailability indicator (e.g. `??`) | Epic 1 → Story 1.4 | ✅ Covered |
| FR48 | Operates normally with subset of providers | Epic 1 → Story 1.3 | ✅ Covered |
| FR49 | Per-provider raw data and error details for diagnosis | Epic 5 → Story 5.2 | ✅ Covered |
| FR50 | Auto-restore provider to ok state when source accessible again | Epic 1 → Story 1.3 | ✅ Covered |

### Missing Requirements

None.

### Coverage Statistics

- Total PRD FRs: 50
- FRs covered in epics: 50
- Coverage percentage: **100%**

## UX Alignment Assessment

### UX Document Status

Not found — and correctly so. pakai is a terminal-native background daemon + CLI tool. All user-facing surfaces (tmux string, waybar JSON, `pakai status` output) are text/JSON outputs consumed by existing shell tooling, not UI rendered by pakai itself. No browser, no native window, no interactive TUI in scope for MVP.

### Alignment Issues

None. PRD explicitly classifies this as "background service + CLI control plane" with terminal-native surfaces only. Architecture and epics are fully consistent with this classification.

### Warnings

None. The post-MVP Bubble Tea TUI dashboard and Tauri widget are explicitly out of scope and correctly excluded from all planning artifacts.

## Epic Quality Review

### Epic Structure

All 5 epics deliver user-facing value. No technical milestone epics. No "Setup Database" anti-patterns. Epic independence verified — no forward dependencies, no circular dependencies.

### Story Dependency Analysis

All 17 stories verified. Each story completable using only prior stories in sequence. No forward dependencies detected.

### Acceptance Criteria Quality

105 total ACs across 17 stories. All use Given/When/Then BDD format. All have specific measurable outcomes. Error conditions covered in every story.

### Violations Found

**🔴 Critical:** None

**🟠 Major:** None

**🟡 Minor (2):**

1. **Story 1.4 — auto-spawn empty-cache behavior unspecified.** ACs cover daemon-starts-within-3s and timeout-failure paths. Missing: behavior when auto-spawn succeeds but first poll hasn't fired yet (cache empty). Recommended addition:
   > Given the daemon was just auto-spawned and has not yet completed its first poll cycle, When GET /status returns an empty array, Then pakai tmux prints an empty/placeholder string and exits with code 0 — empty cache is not an error condition.

2. **Story 3.4 — customization propagation timing implicit.** ACs show correct output but don't explicitly state the within-one-poll-cycle propagation guarantee. Covered implicitly by Story 3.1's live reload contract. Low risk — same epic, sequential ordering.

### Greenfield & Starter Template

Story 1.1 correctly covers `go mod init`, CI/CD setup, and `go install` compatibility — all architecture-specified greenfield initialization requirements present.

### Database Creation Timing

No upfront table creation. OpenCode adapter reads an existing user-owned SQLite file read-only. Fully compliant.

## Summary and Recommendations

### Overall Readiness Status

**READY FOR IMPLEMENTATION**

### Critical Issues Requiring Immediate Action

None. No blockers found.

### Minor Issues to Address Before or During Story 1.4

1. **Story 1.4 — add empty-cache AC:** When auto-spawn succeeds but the first poll cycle hasn't fired, `pakai tmux` should treat an empty `/status` response as a valid non-error state. Add one AC to Story 1.4 before implementation begins — this prevents an agent from incorrectly erroring on the initial empty-cache case.

2. **Story 3.4 — add propagation timing note:** Consider adding a cross-reference to Story 3.1's live reload contract in Story 3.4's ACs, or add a single AC: "Given a label change is written and one poll cycle has elapsed, When I run pakai tmux, Then the new label is shown." Low risk either way — both stories are in Epic 3.

### Recommended Next Steps

1. **Fix Story 1.4** — add the empty-cache auto-spawn AC (5-minute edit to `epics.md`)
2. **Run Sprint Planning `[SP]`** — produces the ordered execution plan for implementation agents
3. **Begin Epic 1, Story 1.1** — `[CS] Create Story` → `[VS] Validate Story` → `[DS] Dev Story` → `[CR] Code Review`

### Final Note

This assessment examined 50 FRs, 23 NFRs, 5 epics, 17 stories, and 105 acceptance criteria across PRD, Architecture, and Epics documents. 2 minor issues identified, 0 critical or major issues. The planning artifacts are coherent, complete, and correctly aligned. Implementation can begin after addressing Story 1.4's minor AC gap.

**Assessor:** Implementation Readiness Check — 2026-05-12
