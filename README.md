# AI Dash

Terminal dashboard for AI coding sessions. Shows Claude, Codex, and OpenCode sessions in one place.

Everything runs locally. No cloud, no accounts, no telemetry.

## Features

- Reads sessions from Claude Code, Codex, and OpenCode using native parsers (not heuristics)
- Fuzzy search across all sessions, live as you type
- Filter by tool, project, date range
- Sort by last active, tool, project, or summary -- per table
- Project overview with session counts and tool usage
- Detail pane with tokens, cost, metadata, related sessions
- Subagent detection (Claude subagents, OpenCode parent/child)
- Nerd Font icons when available, Unicode otherwise
- Nord colors
- Resume sessions or start new ones directly from the dashboard

## Install

### Binary

```bash
curl -L https://github.com/adinhodovic/ai-dash/releases/latest/download/ai-dash-linux-amd64 -o ai-dash
chmod +x ai-dash
```

### From source

```bash
git clone https://github.com/adinhodovic/ai-dash.git
cd ai-dash
make build
./ai-dash
```

## Sources

Sessions are auto-discovered from default paths:

| Tool | Path | Format |
|------|------|--------|
| OpenCode | `~/.local/share/opencode/opencode.db` | SQLite |
| Codex | `~/.codex/sessions/` | JSONL |
| Claude Code | `~/.claude/projects/` | JSONL |

Override paths in `~/.config/ai-dash/config.json`:

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

Generate a JSON Schema for your editor:

```bash
ai-dash schema
```

## Keys

### Navigation

| Key | Action |
|-----|--------|
| `↑/k` `↓/j` | Move selection |
| `g` / `G` | Top / bottom |
| `pgup` / `pgdn` | Page up / down |
| `tab` | Switch focus between Sessions and Projects |

### Sessions

| Key | Action |
|-----|--------|
| `o` | Resume session in a new terminal |
| `n` | New session (pick tool, uses selected project) |

### Search and filter

| Key | Action |
|-----|--------|
| `/` | Search (fuzzy, filters live) |
| `f` | Filter by tool |
| `p` | Filter by project (`/` to search within the list) |
| `D` | Cycle date range |
| `a` | Show/hide subagent sessions |
| `c` | Clear all filters and search |

### Sort

| Key | Action |
|-----|--------|
| `s` | Cycle sort field (applies to focused table) |
| `[` / `]` | Cycle sort field |
| `=` | Flip sort direction |

### Other

| Key | Action |
|-----|--------|
| `?` | Help |
| `v` | Toggle detail pane |
| `S` | Source status |
| `w` / `r` | Save / restore filter preset for current project |
| `q` | Quit |

## License

MIT
