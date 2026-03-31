# AI Dash

See what your AI agents have been up to.

![AI Dash](./docs/images/preview.png)

A lightweight terminal browser for multi-agent coding sessions. Reads from Claude Code, Codex, and OpenCode out of the box. Adding new providers is straightforward -- each is a self-contained package implementing a shared interface. Everything stays local.

## What it does

- Parses Claude Code JSONL transcripts, Codex session logs, and the OpenCode SQLite database
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

```bash
curl -L https://github.com/adinhodovic/ai-dash/releases/latest/download/ai-dash-linux-amd64 -o ai-dash
chmod +x ai-dash
```

## Sources

Sessions are discovered from these default paths:

| Tool | Path | Format |
|------|------|--------|
| OpenCode | `~/.local/share/opencode/opencode.db` | SQLite |
| Codex | `~/.codex/sessions/` | JSONL |
| Claude Code | `~/.claude/projects/` | JSONL |

Override in `~/.config/ai-dash/config.json`:

```json
{
  "opencode_path": "/custom/path/opencode.db",
  "codex_path": "/custom/path/config.toml",
  "claude_path": "/custom/path/projects"
}
```

## Configuration

Config file: `~/.config/ai-dash/config.json`

```json
{
  "$schema": "https://raw.githubusercontent.com/adinhodovic/ai-dash/main/config.schema.json",
  "terminal": "ghostty",
  "poll_interval": "10s",
  "max_age": "14d",
  "default_tool": "claude",
  "auto_select_tool": false,
  "nerd_font": null,
  "age_presets": ["1h", "1d", "3d", "7d", "14d", "30d"]
}
```

| Option | What it does | Default |
|--------|-------------|---------|
| `terminal` | Terminal emulator used to open sessions | `$TERMINAL` |
| `poll_interval` | How often sessions reload | `10s` |
| `max_age` | Only show sessions newer than this | `14d` |
| `default_tool` | Pre-selected tool when pressing `n` | none |
| `auto_select_tool` | Skip the tool picker for new sessions | `false` |
| `nerd_font` | Force Nerd Font on/off, `null` auto-detects | auto |
| `age_presets` | Options when cycling with `D` | `1h,1d,3d,7d,14d,30d` |

Add the `$schema` line to get autocompletion in your editor. You can also run `ai-dash schema` to print it.

## Keys

| Key | Action |
|-----|--------|
| `/` | Search |
| `o` | Resume session |
| `n` | New session |
| `f` / `p` | Filter by tool / project |
| `s` | Cycle sort |
| `tab` | Switch focus |
| `?` | Full help |
| `q` | Quit |
