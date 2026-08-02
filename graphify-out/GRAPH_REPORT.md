# Graph Report - pakai  (2026-08-02)

## Corpus Check
- 66 files · ~140,126 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 686 nodes · 1405 edges · 37 communities (33 shown, 4 thin omitted)
- Extraction: 83% EXTRACTED · 17% INFERRED · 0% AMBIGUOUS · INFERRED: 234 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d226656b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 92|Community 92]]
- [[_COMMUNITY_Community 93|Community 93]]
- [[_COMMUNITY_Community 137|Community 137]]
- [[_COMMUNITY_Community 191|Community 191]]
- [[_COMMUNITY_Community 192|Community 192]]
- [[_COMMUNITY_Community 202|Community 202]]
- [[_COMMUNITY_Community 223|Community 223]]
- [[_COMMUNITY_Community 224|Community 224]]
- [[_COMMUNITY_Community 244|Community 244]]
- [[_COMMUNITY_Community 255|Community 255]]
- [[_COMMUNITY_Community 256|Community 256]]
- [[_COMMUNITY_Community 258|Community 258]]

## God Nodes (most connected - your core abstractions)
1. `UsageWindow` - 35 edges
2. `Config()` - 25 edges
3. `RenderWaybar()` - 25 edges
4. `Server` - 21 edges
5. `RenderTmux()` - 21 edges
6. `Hub` - 20 edges
7. `readGolden()` - 19 edges
8. `RefreshLoop` - 18 edges
9. `Provider` - 17 edges
10. `NewServer()` - 16 edges

## Surprising Connections (you probably didn't know these)
- `fetchDirect()` --calls--> `IsProviderEnabled()`  [INFERRED]
  cmd/pakai/main.go → internal/config/config.go
- `newStatusCmd()` --calls--> `RenderStatus()`  [INFERRED]
  cmd/pakai/main.go → internal/renderer/status.go
- `newTmuxCmd()` --calls--> `GetSeparator()`  [INFERRED]
  cmd/pakai/main.go → internal/config/config.go
- `newTmuxCmd()` --calls--> `RenderTmux()`  [INFERRED]
  cmd/pakai/main.go → internal/renderer/tmux.go
- `newWaybarCmd()` --calls--> `GetSeparator()`  [INFERRED]
  cmd/pakai/main.go → internal/config/config.go

## Import Cycles
- None detected.

## Communities (37 total, 4 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.08
Nodes (56): GetProviderLabel(), ensurePIDDir(), pidFilePath(), readPIDFile(), NewServer(), currentMonthMillis(), DB, T (+48 more)

### Community 1 - "Community 1"
Cohesion: 0.10
Nodes (48): Usage, opencodeWorst(), partitionOpencode(), subLabel(), T, TestOpencodeWorst(), TestPartitionOpencode(), TestRenderTmux_OpencodeSubProviders() (+40 more)

### Community 2 - "Community 2"
Cohesion: 0.11
Nodes (19): Client, HealthResponse, Context, Time, Usage, IsStale(), New(), PIDFilePath() (+11 more)

### Community 3 - "Community 3"
Cohesion: 0.09
Nodes (33): CancelFunc, Hub, Flusher, Int32, T, TestHealthHandlerResponseShape(), TestHealthHandlerStatusOk(), TestHealthHandlerUptimeSeconds() (+25 more)

### Community 4 - "Community 4"
Cohesion: 0.31
Nodes (11): Detection, homeFunc, Detect(), fileExists(), resolveOpenCodeDB(), T, TestDetect_ClaudeOnly(), TestDetect_Found() (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (8): Common pattern, Filtering what's shown, Output Surfaces, Configuration, Output format, Result, Setup, tmux

### Community 6 - "Community 6"
Cohesion: 0.09
Nodes (21): Cache, CacheEntry, blockingProvider, RefreshLoop, Group, DefaultCache(), Time, Usage (+13 more)

### Community 7 - "Community 7"
Cohesion: 0.31
Nodes (7): Context, Time, Usage, New(), NewWithPath(), entry, Provider

### Community 8 - "Community 8"
Cohesion: 0.08
Nodes (55): fetchDirect(), fetchUsages(), Time, Usage, main(), newCompletionCmd(), newConfigCmd(), newConfigGetCmd() (+47 more)

### Community 9 - "Community 9"
Cohesion: 0.08
Nodes (44): AppConfig, DaemonConfig, DisplayConfig, ProviderConfig, ThresholdConfig, Watcher, WidgetConfig, widgetConfigResponse (+36 more)

### Community 11 - "Community 11"
Cohesion: 0.25
Nodes (8): Configuration, Enable / setup, OpenCode (local), Overview, Prerequisites, Result, Sub-providers, Troubleshooting

### Community 12 - "Community 12"
Cohesion: 0.29
Nodes (7): Claude, Configuration, Enable / setup, Overview, Prerequisites, Result, Troubleshooting

### Community 13 - "Community 13"
Cohesion: 0.29
Nodes (7): Configuration, Enable / setup, OpenAI Codex, Overview, Prerequisites, Result, Troubleshooting

### Community 14 - "Community 14"
Cohesion: 0.29
Nodes (7): Configuration, Enable / setup, OpenCode Go (web dashboard), Overview, Prerequisites, Result, Troubleshooting

### Community 15 - "Community 15"
Cohesion: 0.29
Nodes (7): Configuration, Enable / setup, Overview, Pi, Prerequisites, Result, Troubleshooting

### Community 16 - "Community 16"
Cohesion: 0.29
Nodes (6): Configuration, Features, Keybindings, Result, TUI Dashboard, Usage

### Community 17 - "Community 17"
Cohesion: 0.29
Nodes (6): DankMaterialShell (DMS), Features, Install, Plugin configuration (via DMS Settings), Screenshots, Troubleshooting

### Community 18 - "Community 18"
Cohesion: 0.33
Nodes (6): CLI status, Configuration, JSON output, Output format, Result, Usage

### Community 19 - "Community 19"
Cohesion: 0.33
Nodes (5): Configuration, Features, Install, KDE Plasma Widget, Screenshot

### Community 20 - "Community 20"
Cohesion: 0.33
Nodes (6): Configuration, CSS classes, Output format, Screenshot, Setup, Waybar

### Community 37 - "Community 37"
Cohesion: 0.17
Nodes (19): Cmd, coloredProgressBar(), fetchHealth(), Time, Usage, NewDashboardModel(), renderDashboardRowIndented(), renderOpencodeDashboardSection() (+11 more)

### Community 53 - "Community 53"
Cohesion: 0.14
Nodes (10): mockPayload, Server, filterHiddenProviders(), Context, Provider, Request, ResponseWriter, Time (+2 more)

### Community 67 - "Community 67"
Cohesion: 0.07
Nodes (36): cachedUsageWindows, oauthCredentials, Provider, usageAPIResponse, usageWindowResponse, cloneWindows(), Context, Mutex (+28 more)

### Community 68 - "Community 68"
Cohesion: 0.19
Nodes (14): cachedUsageWindows, oauthCredentials, Provider, rateLimitResponse, usageAPIResponse, windowResponse, cloneWindows(), Context (+6 more)

### Community 92 - "Community 92"
Cohesion: 0.19
Nodes (15): MemCache, memEntry, T, TestMemCacheAllEntries(), TestMemCacheDefaultTTL(), TestMemCacheMissingKey(), TestMemCacheMultipleKeys(), TestMemCacheOverwrite() (+7 more)

### Community 93 - "Community 93"
Cohesion: 0.20
Nodes (10): Commands, Configuration, Development, Features, How it works, License, Output surfaces, PakAI (`pakai`) (+2 more)

### Community 137 - "Community 137"
Cohesion: 0.31
Nodes (12): Time, Usage, renderOpencodeSection(), RenderStatus(), renderUsageLine(), renderUsageLineAs(), T, TestRenderStatus_ErrorLine() (+4 more)

### Community 191 - "Community 191"
Cohesion: 0.39
Nodes (7): T, TestWriteUnit(), TestWriteUnit_AutoCreateDir(), TestWriteUnit_Overwrite(), UnitPath(), WriteUnit(), homeFunc

### Community 192 - "Community 192"
Cohesion: 0.22
Nodes (8): AGENTS.md — AI Coding Guidelines for pakai, Analyzing the codebase (Graphify), Architecture, Code conventions, Development commands, Key patterns, Project overview, Tech stack

### Community 202 - "Community 202"
Cohesion: 0.39
Nodes (7): EvaluateThreshold(), Time, DebugInfo, State, ThresholdConfig, ThresholdZone, Usage

### Community 223 - "Community 223"
Cohesion: 0.53
Nodes (5): T, TestStateConstants(), TestUsageNilPointers(), TestUsageStruct(), TestUsageZeroTime()

### Community 224 - "Community 224"
Cohesion: 0.53
Nodes (5): T, TestCachePath(), TestFetch_APIWindows(), TestFetch_Error(), TestID()

### Community 244 - "Community 244"
Cohesion: 0.67
Nodes (3): T, TestFetchAllAggregatesPiSessionProviders(), TestFetchAllHidesProvidersWithoutCurrentMonthUsage()

## Knowledge Gaps
- **86 isolated node(s):** `github.com/dhanifudin/pakai`, `ThresholdConfig`, `mockPayload`, `mockData`, `Provider` (+81 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `UsageWindow` connect `Community 67` to `Community 0`, `Community 2`, `Community 68`?**
  _High betweenness centrality (0.157) - this node is a cross-community bridge._
- **Why does `NewServer()` connect `Community 0` to `Community 3`, `Community 6`, `Community 8`, `Community 9`, `Community 53`?**
  _High betweenness centrality (0.152) - this node is a cross-community bridge._
- **Why does `fetchOpenAIUsageWindows()` connect `Community 0` to `Community 67`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **Are the 10 inferred relationships involving `Config()` (e.g. with `newConfigListCmd()` and `.adaptiveInterval()`) actually correct?**
  _`Config()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 20 inferred relationships involving `RenderWaybar()` (e.g. with `newWaybarCmd()` and `runLivePreview()`) actually correct?**
  _`RenderWaybar()` has 20 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/dhanifudin/pakai`, `ThresholdConfig`, `mockPayload` to the rest of the system?**
  _86 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.0764423076923077 - nodes in this community are weakly interconnected._