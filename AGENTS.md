# AGENTS.md

Guidance for agents working in `ai-dash`.

## Project Shape

- Binary entrypoint: `cmd/ai-dash` (cobra CLI)
- Core session model/loading: `internal/session`
- Configuration: `internal/config` (viper + cobra, JSON config file + env vars)
- Source discovery and importers: `internal/sources`
- Per-tool parsers live in:
  - `internal/sources/codex` (JSONL files)
  - `internal/sources/opencode` (SQLite database)
  - `internal/sources/claude` (JSON/JSONL files)
- Shared source contracts/helpers: `internal/sources/shared`
- Preset persistence: `internal/presets`
- TUI code: `internal/ui` split by concern:
  - `model.go` — Model struct, Init, Update, message dispatch, focus cycling, session filtering
  - `view.go` — View rendering, pane layout, overlays, footer, top bar
  - `sessions.go` — session table resize/sync, open externally
  - `details.go` — detail table, related table resize/sync
  - `overview.go` — overview/projects table resize
  - `picker.go` — filter picker (bubbles list), filter application, presets
  - `keys.go` — keybindings (keyMap), table factory
  - `style.go` — centralized color palette and all styles
  - `util.go` — formatting helpers, path cleaning, time formatting

## Current Direction

- Prefer native parsers over heuristics when real local file formats are known.
- Keep provider-specific parsing logic inside each tool package, not in shared helpers.
- Each provider implements `ResumeArgs(sessionID)` for opening sessions in a terminal.
- Prefer OSS Bubble Tea v2 ecosystem components over hand-rolled widgets.
- Preserve the app as local-first and terminal-first.
- Use `cobra` for CLI and `viper` for configuration.
- Use `humanize` for time/number formatting, `lo` for slice utilities.

## TUI Conventions

### Component-first rendering
- Use `charm.land/bubbletea/v2` for runtime. `View()` returns `tea.View` with `AltScreen: true`.
- Prefer `charm.land/bubbles/v2` components (`table`, `list`, `textinput`, `help`) before building custom UI.
- Let bubbles components own their state: scrolling, cursor, viewport. Do NOT manually manage cursor, call `SetCursor`, or `SetRows` on every frame — only when data actually changes.
- Do not build manual height/width arithmetic to size components. Pass the available height and let the component handle its own chrome (headers, borders, scrolling).

### Layout composition
- Use `lipgloss.JoinVertical` and `lipgloss.JoinHorizontal` for all layout composition. Never use `+ "\n" +` to stitch rendered strings.
- Use `lipgloss.Place` for centering overlays.
- Use `Margin`, `Padding`, `MarginBottom`, `MarginTop` for spacing — never manual `"\n\n"` or `"  "`.
- Do NOT build custom clamp/truncation functions. Trust alt screen mode and component defaults.

### Charm component usage
- `help.Model` — use for footer key hints. Keep `ShortHelp()` to 6-8 bindings max so it fits one line. Put everything else in `FullHelp()`.
- `table.Model` — use for all tabular data. Let the table own its cursor and scrolling state.
- `list.Model` — use for filter pickers with built-in fuzzy search.
- `viewport.Model` — use for scrollable content panes instead of manual line slicing.
- `lipgloss.Height()` / `lipgloss.Width()` — use to measure rendered strings. Never count `\n` manually.
- `lipgloss.NewStyle().Height(n)` — use to force a rendered block to a fixed height (pads or truncates). Never use `strings.Repeat("\n", ...)` for padding.
- `lipgloss.Place()` / `lipgloss.PlaceVertical()` / `lipgloss.PlaceHorizontal()` — use for alignment and centering.
- `style.Margin*()` / `style.Padding*()` — use for spacing between elements. Never use manual `"\n\n"` or `"  "`.

### Styling
- All colors and styles are centralized in `style.go`. Never hardcode `lipgloss.Color(...)` outside `style.go`.
- Use `tableStyles()` for consistent table appearance, `applyHelpStyles()` for help component styling.
- Overlays use `m.styles.overlay` for consistent bordered popups. Popups are mutually exclusive.
- Footer uses `help.ShortHelpView()` for the key bar.

### Layout structure
- Projects pane (full width, 30% height) on top, sessions (60%) + details (40%) below.
- Filter pickers use `list.Model` with built-in fuzzy filtering for project picker, no filtering for tool/status.
- Vim-style navigation: ↑/k, ↓/j for movement, g/G for top/bottom. Add new keybindings to both the keyMap struct and FullHelp output.

## Source Import Rules

- Start from documented config/storage roots first.
- Keep discovery separate from parsing/import.
- Put format-aware parsing in the relevant provider package.
- Each provider implements `SessionProvider` interface: `Name()`, `Discover()`, `ImportSessions()`, `ResumeArgs()`. Add a compile-time check: `var _ shared.SessionProvider = Source{}`.
- Providers that can identify parent-child session relationships implement the optional `SubagentClassifier` interface (`ParentSessionID(s session.Session) string`). The discovery layer applies classification after collecting sessions. Each tool has its own conventions (Claude: `subagents/` directory, OpenCode: `parent_id` DB column).
- Opencode reads from SQLite (`~/.local/share/opencode/opencode.db`), not JSON files.
- Fallback priority: native parser -> provider-specific structured fallback -> shared heuristic importer -> bundled sample data.
- Never let bundled sample data override discovered real sessions.

## Configuration

- Config file: `~/.config/ai-dash/config.json` (viper auto-discovery)
- Env vars override file values with `AIDASH_` prefix (e.g. `AIDASH_MAX_AGE`, `AIDASH_POLL_INTERVAL`)
- `$TERMINAL` env var controls which terminal opens sessions (e.g. `ghostty`, `kitty`)
- `max_age` supports day shorthand (`14d`) in addition to Go durations (`336h`)
- `./ai-dash schema` generates JSON Schema for the config file

## Fixture Anonymization

- Preserve the real schema and field relationships, but replace user/project-specific values.
- Rewrite absolute paths to neutral examples unless the path shape itself is under test.
- Replace IDs, slugs, titles, and prompts with safe stand-ins.
- Avoid storing secrets, API keys, internal hostnames, or personal names in fixtures.

## Testing And Validation

- Run `make fmt`, `make build`, and `make test` after meaningful changes.
- Add provider-specific tests when touching importer/parser logic.
- Opencode tests create an in-memory SQLite database via `createTestDB()`.
- UI tests use `resize()` and `sendKey()` helpers, check `View().Content` for output.
- Prefer anonymized real-format fixtures over invented formats.

## Formatting Helpers

- Use `formatCost`, `formatTokens`, `formatCount`, `durationLabel`, `timeAgo` from `util.go`.
- `formatCount` uses `humanize.Comma` for comma-separated numbers.
- `timeAgo` uses `humanize.Time` for relative timestamps.
- `cleanProjectName` converts slug-encoded paths to readable `~/path/form`.
- `shortenPath` replaces the user's home directory with `~` in raw paths.

## Style Notes

- Follow existing Go package boundaries; avoid unnecessary new top-level packages.
- Keep comments sparse and only where they clarify non-obvious behavior.
- Prefer small helpers and standard library code where it keeps parsing logic clearer.
- Do not introduce destructive behavior or live external dependencies.
