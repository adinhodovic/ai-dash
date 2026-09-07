#!/usr/bin/env python3
"""Generates fake ai-dash session data (claude/codex/opencode formats) for
recording a demo GIF, entirely under a throwaway fake $HOME."""
import json
import os
import sqlite3
import sys
import uuid
from datetime import datetime, timedelta, timezone

FAKE_HOME = sys.argv[1]
NOW = datetime.now(timezone.utc)


def ts(minutes_ago=0, hours_ago=0, days_ago=0):
    return (NOW - timedelta(minutes=minutes_ago, hours=hours_ago, days=days_ago)).strftime(
        "%Y-%m-%dT%H:%M:%S.000Z"
    )


def ms_ago(minutes_ago=0, hours_ago=0, days_ago=0):
    return int((NOW - timedelta(minutes=minutes_ago, hours=hours_ago, days=days_ago)).timestamp() * 1000)


# ---------------------------------------------------------------------------
# Claude Code: ~/.claude/projects/<encoded-project>/<session-id>.jsonl
# ---------------------------------------------------------------------------
def write_claude_session(project_dir_name, session_id, cwd, branch, summary, model,
                          stop_reason, minutes_ago, extra_minutes_started=6):
    base = os.path.join(FAKE_HOME, ".claude", "projects", project_dir_name)
    os.makedirs(base, exist_ok=True)
    path = os.path.join(base, f"{session_id}.jsonl")
    started = ts(minutes_ago=minutes_ago + extra_minutes_started)
    ended = ts(minutes_ago=minutes_ago)
    lines = [
        {
            "type": "user",
            "sessionId": session_id,
            "slug": summary[:40],
            "cwd": cwd,
            "gitBranch": branch,
            "timestamp": started,
            "message": {"role": "user", "content": summary},
        },
        {
            "type": "assistant",
            "sessionId": session_id,
            "timestamp": ended,
            "message": {
                "role": "assistant",
                "model": model,
                "stop_reason": stop_reason,
                "content": [{"type": "text", "text": "Working on it."}],
                "usage": {"input_tokens": 2400, "output_tokens": 860},
            },
        },
    ]
    with open(path, "w") as f:
        for line in lines:
            f.write(json.dumps(line) + "\n")


# ---------------------------------------------------------------------------
# Codex: ~/.codex/sessions/rollout-<ts>-<uuid>.jsonl
# ---------------------------------------------------------------------------
def write_codex_session(cwd, summary, model, minutes_ago, event_type,
                         extra_minutes_started=6, cli_version="0.5.0"):
    base = os.path.join(FAKE_HOME, ".codex", "sessions")
    os.makedirs(base, exist_ok=True)
    sid = str(uuid.uuid4())
    started = ts(minutes_ago=minutes_ago + extra_minutes_started)
    ended = ts(minutes_ago=minutes_ago)
    path = os.path.join(base, f"rollout-{started[:19].replace(':', '-')}-{sid}.jsonl")
    lines = [
        {
            "timestamp": started,
            "type": "session_meta",
            "payload": {
                "id": sid, "timestamp": started, "cwd": cwd,
                "cli_version": cli_version, "model_provider": "openai",
            },
        },
        {
            "timestamp": started,
            "type": "turn_context",
            "payload": {"cwd": cwd, "model": model, "effort": "medium"},
        },
        {
            "timestamp": started,
            "type": "response_item",
            "payload": {
                "type": "message", "role": "user",
                "content": [{"type": "input_text", "text": summary}],
            },
        },
        {
            "timestamp": ended,
            "type": "event_msg",
            "payload": {"type": event_type},
        },
    ]
    with open(path, "w") as f:
        for line in lines:
            f.write(json.dumps(line) + "\n")


# ---------------------------------------------------------------------------
# OpenCode: sqlite db with session/message/part tables
# ---------------------------------------------------------------------------
def init_opencode_db(path):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    conn = sqlite3.connect(path)
    conn.execute("""
        CREATE TABLE session (
            id TEXT PRIMARY KEY, parent_id TEXT, slug TEXT, directory TEXT,
            title TEXT, version TEXT,
            summary_additions INTEGER, summary_deletions INTEGER, summary_files INTEGER,
            time_created INTEGER, time_updated INTEGER
        )
    """)
    conn.execute("""
        CREATE TABLE message (
            id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, data TEXT
        )
    """)
    conn.execute("""
        CREATE TABLE part (
            id TEXT PRIMARY KEY, message_id TEXT, time_created INTEGER, data TEXT
        )
    """)
    conn.commit()
    return conn


def write_opencode_session(conn, directory, title, model, minutes_ago,
                            finish, part_type="", part_status="",
                            extra_minutes_started=6, completed=True):
    sid = str(uuid.uuid4())
    created = ms_ago(minutes_ago=minutes_ago + extra_minutes_started)
    updated = ms_ago(minutes_ago=minutes_ago)
    conn.execute(
        "INSERT INTO session (id, parent_id, slug, directory, title, version, "
        "summary_additions, summary_deletions, summary_files, time_created, time_updated) "
        "VALUES (?, '', ?, ?, ?, ?, 12, 4, 3, ?, ?)",
        (sid, title[:30], directory, title, "0.9.0", created, updated),
    )
    mid = str(uuid.uuid4())
    msg_data = {
        "role": "assistant",
        "model": {"modelID": model},
        "finish": finish,
        "time": {"completed": updated if completed else 0},
    }
    conn.execute(
        "INSERT INTO message (id, session_id, time_created, data) VALUES (?, ?, ?, ?)",
        (mid, sid, updated, json.dumps(msg_data)),
    )
    if part_type:
        pid = str(uuid.uuid4())
        part_data = {"type": part_type, "state": {"status": part_status}}
        conn.execute(
            "INSERT INTO part (id, message_id, time_created, data) VALUES (?, ?, ?, ?)",
            (pid, mid, updated, json.dumps(part_data)),
        )


def main():
    # --- Claude ---
    write_claude_session(
        "-home-demo-ai-dash", str(uuid.uuid4()),
        "/home/demo/ai-dash", "main",
        "review the new attention detection logic and check the staleness window feels right",
        "claude-sonnet-5", "pause_turn", minutes_ago=3,
    )
    write_claude_session(
        "-home-demo-ai-dash", str(uuid.uuid4()),
        "/home/demo/ai-dash", "feat/multi-select-filters",
        "wire up the multi-select project filter picker",
        "claude-sonnet-5", "tool_use", minutes_ago=0,
    )
    write_claude_session(
        "-home-demo-billing-service", str(uuid.uuid4()),
        "/home/demo/billing-service", "fix/invoice-rounding",
        "investigate why invoice totals round differently in the reporting export",
        "claude-sonnet-5", "max_tokens", minutes_ago=8,
    )
    write_claude_session(
        "-home-demo-growth-analytics", str(uuid.uuid4()),
        "/home/demo/growth-analytics", "main",
        "add cohort retention chart to the weekly dashboard",
        "claude-sonnet-5", "end_turn", minutes_ago=120,
    )
    write_claude_session(
        "-home-demo-ai-dash", str(uuid.uuid4()),
        "/home/demo/ai-dash", "main",
        "fix flaky test in session attention detection suite",
        "claude-sonnet-5", "end_turn", minutes_ago=60 * 24,
    )
    write_claude_session(
        "-home-demo-growth-analytics", str(uuid.uuid4()),
        "/home/demo/growth-analytics", "main",
        "draft rfc for event schema versioning",
        "claude-sonnet-5", "end_turn", minutes_ago=60 * 24 * 3,
    )

    # --- Codex ---
    write_codex_session(
        "/home/demo/billing-service",
        "clean up the pagination query needed for the transaction ledger export",
        "gpt-5-codex", minutes_ago=50, event_type="task_started",
    )
    write_codex_session(
        "/home/demo/docs-site",
        "update the getting started guide screenshots",
        "gpt-5-codex", minutes_ago=60 * 6, event_type="task_complete",
    )
    write_codex_session(
        "/home/demo/billing-service",
        "add retry logic for webhook delivery failures",
        "gpt-5-codex", minutes_ago=60 * 24, event_type="turn_aborted",
    )
    write_codex_session(
        "/home/demo/ai-dash",
        "port codex source parser tests to table-driven style",
        "gpt-5-codex", minutes_ago=60 * 24 * 4, event_type="task_complete",
    )

    # --- OpenCode ---
    db_path = os.path.join(FAKE_HOME, ".local", "share", "opencode", "opencode.db")
    conn = init_opencode_db(db_path)
    write_opencode_session(
        conn, "/home/demo/docs-site",
        "restructure sidebar navigation for the new docs sections",
        "gpt-5", minutes_ago=1, finish="", part_type="tool", part_status="running",
        completed=False,
    )
    write_opencode_session(
        conn, "/home/demo/docs-site",
        "write migration guide from v1 config to v2",
        "gpt-5", minutes_ago=5, finish="stop",
    )
    write_opencode_session(
        conn, "/home/demo/growth-analytics",
        "benchmark postgres vs clickhouse for the events table",
        "gpt-5", minutes_ago=60 * 24 * 9, finish="tool-calls",
    )
    write_opencode_session(
        conn, "/home/demo/ai-dash",
        "spike: opencode multi-select filter UX",
        "gpt-5", minutes_ago=60 * 24 * 12, finish="tool-calls",
    )
    conn.commit()
    conn.close()

    # VHS's headless renderer has no Nerd Font available, so force the
    # plain-ASCII icon fallback for this fake $HOME rather than relying on
    # whatever the recording machine happens to have installed.
    config_dir = os.path.join(FAKE_HOME, ".config", "ai-dash")
    os.makedirs(config_dir, exist_ok=True)
    with open(os.path.join(config_dir, "config.json"), "w") as f:
        json.dump({"nerd_font": False}, f)

    print("Fake demo data written under", FAKE_HOME)


if __name__ == "__main__":
    main()
