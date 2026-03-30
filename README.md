# ai-dash

Local-first TUI experiment for viewing AI coding sessions across OpenCode, Codex, and Claude in one place.

## Why Go

Go is a strong fit for TUIs when you want something fast to prototype, easy to ship as a single binary, and pleasant to maintain. The Bubble Tea ecosystem gives you solid primitives for layout, keyboard handling, and styling.

This project now leans on a few OSS TUI building blocks from Charmbracelet:

- `bubbletea` for the app runtime
- `bubbles/textinput` for search editing
- `bubbles/viewport` for scrollable detail panes
- `bubbles/help` for discoverable key hints
- `lipgloss` for styling

If you want the strongest alternatives:

- Go + Bubble Tea: fastest path to a polished CLI/TUI MVP
- Rust + ratatui: excellent performance and control, steeper iteration cost
- Python + Textual: productive and expressive, weaker single-binary distribution story

## MVP

This first pass includes:

- a simple session model shared across tools
- a TUI split into a session list and details pane
- support for loading data from `sessions.sample.json` or `sessions.json`
- lightweight summaries by tool, project, status, and token usage
- a discovery layer for documented local `OpenCode`, `Codex`, and `Claude Code` config/session roots

## Run

```bash
make tidy
go run ./cmd/ai-dash
```

## Build

```bash
make build
./ai-dash
```

## Documented Local Sources

The current discovery layer follows documented/default local paths first and supports env overrides:

- `OpenCode`: `~/.config/opencode/opencode.json`, `~/.config/opencode/tui.json`
- `Codex`: `~/.codex/config.toml`
- `Claude Code`: `~/.claude.json`, `~/.claude/settings.json`, `~/.claude/projects`

Supported overrides:

- `AIDASH_OPENCODE_CONFIG`
- `AIDASH_OPENCODE_TUI_CONFIG`
- `AIDASH_CODEX_CONFIG`
- `AIDASH_CLAUDE_STATE`
- `AIDASH_CLAUDE_SETTINGS`
- `AIDASH_CLAUDE_PROJECTS_DIR`
- `AIDASH_PRESETS_PATH`

The importer layer now does three things:

- parses Claude transcript files under `~/.claude/projects`
- scans `~/.codex` for likely session/history files and imports them with heuristics
- scans `~/.config/opencode` for likely session/history files and imports them with heuristics

Docs check note:

- Codex docs clearly document config at `~/.codex/config.toml` and a configurable `log_dir`, but do not clearly document a canonical session-history storage schema.
- OpenCode docs clearly document config locations like `~/.config/opencode/opencode.json`, `~/.config/opencode/tui.json`, project `opencode.json`, and `.opencode/`, but do not clearly document a canonical local session-history schema.

So OpenCode and Codex still use safe heuristics for now, while Claude uses a stronger transcript import path.

The source layer is organized per tool under:

- `internal/sources/opencode`
- `internal/sources/codex`
- `internal/sources/claude`

with shared discovery/import interfaces and result types in `internal/sources/shared`.

## Controls

- `tab` / `shift+tab`: cycle focus between list, filters, and search
- `j` / `k` or arrow keys: move selection
- `/`: start search mode
- `enter`: apply search
- `esc`: leave search mode
- `f` / `F`: next or previous tool filter
- `s` / `S`: next or previous status filter
- `p` / `P`: next or previous project filter
- `w`: save the current filter/search preset for the selected project
- `r`: restore the saved preset for the selected project
- `v`: collapse or expand the detail pane
- arrow keys / `pgup` / `pgdown`: scroll the details pane when it is open
- `pgup` / `pgdown`: scroll the detail viewport
- `c`: clear all filters
- `q` or `ctrl+c`: quit

## Session Schema

```json
{
  "sessions": [
    {
      "id": "sess_001",
      "tool": "opencode",
      "project": "ai-dash",
      "repo": "/home/adin/oss/ai-dash",
      "branch": "main",
      "status": "active",
      "started_at": "2026-03-29T14:00:00Z",
      "ended_at": "2026-03-29T14:42:00Z",
      "model": "gpt-5.4",
      "summary": "Scaffolded a local-first multi-agent dashboard TUI.",
      "transcript_path": "/tmp/opencode-session.md",
      "tokens_in": 4200,
      "tokens_out": 2100,
      "cost_usd": 0.22,
      "tags": ["mvp", "dashboard"]
    }
  ]
}
```

## Next Steps

1. Replace Codex/OpenCode heuristic import with real format-aware parsers.
2. Add live updates with file watching.
3. Add richer list widgets and multi-select/filter chips.
4. Add a small HTTP API so the same data can power a web dashboard later.
