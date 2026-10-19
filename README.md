# AI Dash

![AI Dash](./docs/images/demo.gif)

A terminal UI for browsing coding sessions across multiple tools from local transcripts, session logs, and databases. It currently supports Claude Code, Codex, OpenCode, and pi.

## What it does

- Parses Claude Code JSONL transcripts, Codex session logs, OpenCode's SQLite database, and pi session JSONL files
- Fuzzy search across sessions, live as you type
- Filter by tool, project, or date range
- Sort per table (last active, tool, project, summary)
- Project overview with session counts and tool breakdown
- Detail pane with tokens, cost, metadata, related sessions
- Picks up subagent/child sessions (Claude subagents, OpenCode parent/child)
- Nerd Font icons when available, falls back to Unicode
- Resume or start sessions from the dashboard

## Install

### From source

```bash
git clone https://github.com/adinhodovic/ai-dash.git
cd ai-dash
make build
./ai-dash
```

### Binary

Pre-built binaries are available on the [releases page](https://github.com/adinhodovic/ai-dash/releases).

**Linux:**

```bash
curl -L https://github.com/adinhodovic/ai-dash/releases/latest/download/ai-dash-linux-amd64 -o ai-dash
chmod +x ai-dash
```

**macOS:**

```bash
curl -L https://github.com/adinhodovic/ai-dash/releases/latest/download/ai-dash-darwin-arm64 -o ai-dash
chmod +x ai-dash
```

For other architectures, see all binaries on the [releases page](https://github.com/adinhodovic/ai-dash/releases).

### Updating

Release binaries can update themselves:

```bash
ai-dash update          # download and install the latest release
ai-dash update --check  # only report whether an update is available
```

The download is verified against the release's `checksums.txt`. Builds from source print the `go install` command to use instead; pass `--force` to replace them with the release binary.

## Configuration

Config file: `~/.config/ai-dash/config.json`

### Sources

Sessions are discovered from default paths. Override them if needed:

| Tool | Default path | Config key |
|------|-------------|------------|
| OpenCode | `~/.local/share/opencode/opencode.db` or `~/Library/Application Support/opencode/opencode.db` on macOS | `opencode_path` |
| Codex | `~/.codex/config.toml` | `codex_path` |
| Claude Code | `~/.claude/projects/` | `claude_path` |
| pi | `~/.pi/agent/sessions/` | `pi_path` |

Parsed session metadata is cached in `~/.cache/ai-dash/sessions.db` (`~/Library/Caches/ai-dash/sessions.db` on macOS). A transcript is reread only when its size or mtime changes. Delete the file to force a full rescan.

### Options

```json
{
  "$schema": "https://raw.githubusercontent.com/adinhodovic/ai-dash/main/config.schema.json",
  "terminal": "ghostty",
  "multiplexer": "auto",
  "tmux_mode": "terminal",
  "poll_interval": "10s",
  "default_age_filter": "14d",
  "default_tool": "claude",
  "auto_select_tool": false,
  "nerd_font": null,
  "age_presets": ["1h", "1d", "3d", "7d", "14d", "30d"]
}
```

| Option | What it does | Default |
|--------|-------------|---------|
| `terminal` | Terminal emulator used to open/resume sessions (e.g. `ghostty`, `kitty`, `alacritty`, `wezterm`, `foot`) | `$TERMINAL`, else the first of ghostty, kitty, alacritty, wezterm, foot, gnome-terminal, konsole, xfce4-terminal, x-terminal-emulator, xterm found on `PATH` |
| `multiplexer` | Open sessions in tmux/zellij when ai-dash runs inside one. `auto` detects via `$TMUX`/`$ZELLIJ`; `tmux`/`zellij` force one; `off` disables. Falls back to `terminal` when no multiplexer is active. | `auto` |
| `tmux_mode` | How sessions open under tmux: `terminal` opens a new `terminal` window running the session in its own tmux session (falls back to `window` when `terminal` is unset), `window` opens a new tmux window and switches to it, `background` opens one without switching. | `terminal` |
| `poll_interval` | How often sessions reload | `10s` |
| `default_age_filter` | Default age filter used on load and when clearing filters | `14d` |
| `default_tool` | Pre-selected tool when pressing `n` | none |
| `auto_select_tool` | Skip the tool picker for new sessions | `false` |
| `nerd_font` | Force Nerd Font on/off, `null` auto-detects | auto |
| `age_presets` | Options when cycling with `D` | `1h,1d,3d,7d,14d,30d` |

Inside tmux a new session opens in a new terminal window running its own tmux session (see `tmux_mode`); inside zellij it opens
as a new pane via `zellij run`. Set `multiplexer: "off"` to always spawn a
fresh terminal emulator instead.

Add the `$schema` line to get autocompletion in your editor. You can also run `ai-dash schema` to print it.

On macOS, `terminal` should be a CLI terminal binary like `ghostty` or `kitty`.

### Actions

Actions open the selected session's tool with a prompt already submitted. Press `x` to pick one, or bind it to a key. `generic` actions apply to every tool; `claude`, `codex`, `opencode` and `pi` actions apply to that tool only and take precedence over a generic action with the same key or label.

```json
{
  "actions": {
    "generic": [
      { "key": "ctrl+r", "label": "continue", "prompt": "continue" },
      { "key": "ctrl+t", "label": "test", "prompt": "Run the tests and fix any failures" }
    ],
    "codex": [
      { "key": "ctrl+o", "label": "review", "mode": "new", "prompt": "Review the uncommitted changes in {{.Project}}" }
    ]
  }
}
```

| Field | What it does |
|-------|-------------|
| `label` | Name in the `x` picker. Reusing a built-in label replaces that action. |
| `key` | Optional direct key. Keys ai-dash already uses are ignored. |
| `mode` | `resume` (default) sends the prompt to the selected session; `new` starts a session in its project. |
| `prompt` | Go template with `{{.Input}}`, `{{.Project}}`, `{{.SessionID}}`, `{{.Tool}}`, `{{.Summary}}`, `{{.Model}}`. |
| `input_label` | A prompt that uses `{{.Input}}` asks for text first; this labels the field. |
| `input_optional` | Run even when the input is left empty. |
| `disabled` | Remove the built-in action with the same label. |

`?` lists the generic actions, then each tool's own. Actions without a key are listed under `x`.

Built-in actions:

| Scope | Key | Label | Prompt |
|-------|-----|-------|--------|
| generic | `ctrl+r` | continue | `continue` |
| generic | `ctrl+e` | message | asks for a message and sends it |
| generic | `ctrl+n` | new with prompt | asks for a prompt and starts a new session |
| claude | | artifact | asks what to summarize and creates an artifact of it |

## Keys

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move selection |
| `g`/`G` | Top / bottom |
| `pgup`/`pgdn` | Page up / down |
| `r` | Resume session |
| `R` | Rename session |
| `n` | New session |
| `x` | Run an action on the session |
| `ctrl+r` / `ctrl+e` / `ctrl+n` | Continue / send a message / new session with a prompt (see [Actions](#actions)) |
| `v` | Toggle detail pane |
| `i` | Toggle IDs & metadata |
| `/` | Search |
| `t` / `p` | Filter by tool / project |
| `D` | Cycle age range |
| `a` | Toggle subagents |
| `!` | Toggle needs-attention filter |
| `A` | Toggle active-only filter |
| `c` | Clear filters/search |
| `s` | Cycle sort field |
| `[`/`]` | Cycle sort field |
| `=` | Toggle sort direction |
| `S` | Toggle sources view |
| `?` | Full help |
| `q` | Quit |

Press `?` in the app to see the full context-aware keymap.

## Development

### Common tasks

```bash
make fmt
make build
make test
golangci-lint run ./...
```

Run the app locally with:

```bash
./ai-dash
```

### Project layout

- `cmd/ai-dash` - cobra entrypoint
- `internal/config` - config loading and schema generation
- `internal/session` - shared session model and sorting
- `internal/sources/claude` - Claude Code parser
- `internal/sources/codex` - Codex parser
- `internal/sources/opencode` - OpenCode parser
- `internal/sources/pi` - pi parser
- `internal/sources/shared` - shared source discovery helpers
- `internal/ui` - Bubble Tea TUI

### Source support

Only official provider files are supported:

- Claude Code transcripts
- Codex session JSONL files
- OpenCode SQLite database
- pi session JSONL files

Do not add generic or custom session JSON loaders.

### Testing notes

- Add provider-specific tests when changing importer or parser logic
- Keep fixtures under each source package's `testdata/` directory
- Run `make fmt`, `make build`, `make test`, and `golangci-lint run ./...` before finishing changes

## License

Apache License 2.0
