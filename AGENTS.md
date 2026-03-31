# AGENTS.md

Guidance for agents working in `ai-dash`.

## Project Shape

- Binary entrypoint: `cmd/ai-dash` (cobra CLI)
- Core session model/loading: `internal/session`
- Configuration: `internal/config` (viper + cobra, JSON config file + env vars)
- Source discovery and importers: `internal/sources`
- Per-tool native parsers:
  - `internal/sources/claude` — JSONL transcript parser with token/model/status extraction
  - `internal/sources/opencode` — SQLite reader with model extraction from message table
  - `internal/sources/codex` — JSONL log parser with session metadata
- Shared source contracts/helpers: `internal/sources/shared`
- Preset persistence: `internal/presets`
- Icon system: `internal/ui/icon` (Nerd Font with Unicode fallback, auto-detected)
- TUI code: `internal/ui` split by concern:
  - `model.go` — Model struct, Init, Update, message dispatch, focus cycling, filtering, fuzzy search
  - `view.go` — View rendering, pane layout, overlays, top bar, footer, detail items
  - `sessions.go` — Session table, resume session, new session, open transcript
  - `details.go` — Detail table (key-value), related sessions table
  - `overview.go` — Projects table, stats panel, per-table sorting
  - `picker.go` — Filter picker overlay (bubbles list with fuzzy search), Nord-styled
  - `keys.go` — Keybindings (keyMap), context-aware shorthelp, table factories
  - `style.go` — Nord color scheme, all styles centralized
  - `util.go` — Formatting helpers, path cleaning, time formatting, renderPane

## Current Direction

- Native parsers for each tool — no heuristic/generic importers for known formats.
- Each provider implements `SessionProvider` and optionally `SubagentClassifier`.
- Each provider implements `NewSessionArgs(projectDir)` and `ResumeArgs(sessionID)`.
- Source paths configurable via `config.json`, no legacy env var overrides.
- Prefer OSS Bubble Tea v2 ecosystem components over hand-rolled widgets.
- Local-first and terminal-first. No cloud, no HTTP API.
- Nord color scheme throughout. All colors in `style.go`.
- Nerd Font icons auto-detected via `fc-list`, fallback to Unicode. Opt-out via config.
- Use `cobra` for CLI, `viper` for configuration, `sahilm/fuzzy` for search.
- Use `humanize` for time/number formatting, `lo` for slice utilities.

## TUI Conventions

### Component-first rendering
- Use `charm.land/bubbletea/v2` for runtime. `View()` returns `tea.View` with `AltScreen: true`.
- Prefer `charm.land/bubbles/v2` components (`table`, `list`, `textinput`, `help`) before building custom UI.
- Let bubbles components own their state: scrolling, cursor, viewport.
- Call `UpdateViewport()` on tables after `SetRows`/`SetHeight` to ensure rendering is current.
- Do not shadow the table's cursor with manual `selected int` tracking.

### Layout composition
- Use `lipgloss.JoinVertical` and `lipgloss.JoinHorizontal` for all layout composition.
- Use `lipgloss.Place` for centering overlays.
- Use `Margin*()` / `Padding*()` for all spacing — never manual `" "`, `"\n"`, or `strings.Repeat`.
- `lipgloss.Height(n)` sets total height including borders — not content height. Use `Height(h).MaxHeight(h)` on panes to enforce exact dimensions.
- `renderPane()` in `util.go` is the standard bordered panel renderer. All panes use it.

### Charm component usage
- `help.Model` — footer key hints. `shortHelpForFocus()` returns context-aware bindings per focused pane.
- `table.Model` — all tabular data. Column headers include sort indicators via `sortHeader()`.
- `list.Model` — filter pickers. Forward `FilterMatchesMsg` back to the list by passing unhandled messages when picker is active. Check `FilterState()` to avoid intercepting keys during filtering.
- `lipgloss.Height()` / `lipgloss.Width()` — measure rendered strings. Never count `\n` manually.
- `lipgloss.NewStyle().Padding()` — use for spacing. Never use `" "` string concatenation.

### Styling
- Nord color scheme defined in `style.go`. All semantic colors map to `nord*` constants.
- Never hardcode `lipgloss.Color(...)` outside `style.go` (except one-off view-specific styles like the title).
- `tableStyles()` for table appearance, `applyHelpStyles()` for help component.
- Picker delegate styles set in `newPicker()` to match Nord scheme.
- Filter chips use `badge` style (yellow background) with `Padding(0, 1).MarginRight(1)`.

### Layout structure
- Top row: Projects table (70%) + Overview stats (30%), sharing `topPaneHeight`.
- Bottom row: Sessions table (70%) + Details pane (30%), sharing `bottomPaneHeight`.
- Focus cycles between Sessions and Projects only (detail pane is display-only).
- Sorting is per-table: `s` cycles sort for the focused table.

## Source Import Rules

- Each provider implements `SessionProvider`: `Name()`, `Discover()`, `ImportSessions()`, `ResumeArgs()`, `NewSessionArgs()`.
- Add compile-time check: `var _ shared.SessionProvider = Source{}`.
- Optional `SubagentClassifier` interface for parent-child detection. Discovery layer calls it after collecting sessions.
- Source constructors take `config.Config` for configurable paths.
- Claude: native JSONL parser extracts first user message as summary, model from assistant messages, tokens from usage, status from `stop_reason`.
- OpenCode: SQLite with model extracted from `message.data` JSON via `json_extract`. Includes `summary_additions/deletions/files`.
- Codex: JSONL parser with `session_meta`, `turn_context`, `response_item`, `event_msg` types.
- `Session.Meta` (`map[string]string`) stores tool-specific metadata displayed in the detail pane.
- `Session.Project` should be the real path (not slug-encoded). Claude uses `cwd` from transcript.

## Configuration

- Config file: `~/.config/ai-dash/config.json` (viper auto-discovery)
- Env vars: `AIDASH_` prefix auto-bound by viper (e.g. `AIDASH_OPENCODE_PATH`)
- `$TERMINAL` controls which terminal opens sessions
- `max_age` supports day shorthand (`14d`) and Go durations (`336h`)
- `age_presets` configures the `D` key cycle options
- `nerd_font: null` means auto-detect; set `false` to opt out
- `./ai-dash schema` generates JSON Schema for editor autocompletion

## Fixture Anonymization

- Preserve the real schema and field relationships, but replace user/project-specific values.
- Rewrite absolute paths to neutral examples unless the path shape itself is under test.
- Replace IDs, slugs, titles, and prompts with safe stand-ins.
- Avoid storing secrets, API keys, internal hostnames, or personal names in fixtures.
- Use `testdata/` directories within each source package for JSONL fixtures.

## Testing And Validation

- Run `make fmt`, `make build`, and `make test` after meaningful changes.
- Run `golangci-lint run ./...` to check for lint issues (errcheck, gofumpt, golines, unused).
- Add provider-specific tests when touching importer/parser logic.
- OpenCode tests create an in-memory SQLite database with `session` and `message` tables via `createTestDB()`.
- Claude tests use `testdata/session.jsonl` fixture with anonymized real-format JSONL.
- UI tests use `resize()` and `sendKey()` helpers, check `View().Content` for output.
- `TestViewFitsTerminal` verifies layout doesn't exceed terminal bounds at multiple sizes.

## Formatting Helpers

- Use `formatCost`, `formatTokens`, `durationLabel`, `timeAgo` from `util.go`.
- `timeAgo` uses `humanize.Time` for relative timestamps.
- `cleanProjectName` shortens absolute paths via `shortenPath` (`~` substitution). No slug decoding needed since sources now provide real paths.
- `humanizeKey` converts `snake_case` meta keys to `Title Case` for display.

## Style Notes

- Follow existing Go package boundaries; avoid unnecessary new top-level packages.
- Keep comments sparse and only where they clarify non-obvious behavior.
- Prefer small helpers and standard library code where it keeps parsing logic clearer.
- Do not introduce destructive behavior or live external dependencies.
- Line length enforced by `golines`. Break long lines by extracting variables or using multiline function calls.
