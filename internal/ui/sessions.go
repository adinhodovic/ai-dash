package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"

	"github.com/adinhodovic/ai-dash/internal/session"
	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

// attentionLabel describes why a session needs attention and for how long,
// e.g. "waiting 45m". The duration is time since last observed activity —
// the same signal session.Attention uses to derive the reason itself, so it
// doubles as "how long in this state" for every reason, not just stalled.
func attentionLabel(s session.Session) string {
	reason := session.Attention(s)
	if reason == session.AttentionNone {
		return ""
	}
	return fmt.Sprintf("%s %s", reason, ageLabel(time.Since(s.EndedAt)))
}

func (m *Model) resizeTable(filtered []session.Session) {
	width := max(44, m.width)
	if !m.detailCollapsed {
		width = max(40, m.width*70/100)
	}
	// Subtract pane border (2) and interior padding for inner width; sort
	// header + its margin line (2) for height.
	tableW := max(40, width-2-2*uilayout.PanePadding)
	height := max(2, uilayout.PaneBodyHeight(uilayout.BottomPaneHeight(m.height))-1)
	m.sessionViewport.SetWidth(tableW)
	m.sessionViewport.SetHeight(max(1, height-2))
	m.syncTable(filtered)
}

func (m *Model) syncTable(filtered []session.Session) {
	if len(filtered) == 0 {
		m.sessionCursor = 0
		m.sessionViewport.SetContent("")
		return
	}
	m.sessionCursor = max(0, min(len(filtered)-1, m.sessionCursor))
	width := m.sessionViewport.Width()
	blocks := make([]string, 0, len(filtered))
	for i, s := range filtered {
		blocks = append(blocks, renderSessionRow(*m, s, i == m.sessionCursor, width))
	}
	// A faint rule between rows (not just blank space) so one session's
	// boundary is unambiguous, matching sessionRowGap's single line.
	divider := m.styles.Rule.Render(strings.Repeat("─", width))
	m.sessionViewport.SetContent(strings.Join(blocks, "\n"+divider+"\n"))
	m.sessionViewport.SetYOffset(followCursor(
		m.sessionCursor,
		len(filtered),
		m.sessionViewport.Height(),
		m.sessionViewport.YOffset(),
	))
}

func (m *Model) resizeSourceTable() {
	width := max(40, m.width*70/100-6)
	m.sourceTable.SetColumns([]table.Column{
		{Title: theme.Tool + " Tool", Width: 9},
		{Title: theme.Meta + " Format", Width: 9},
		{Title: theme.Active + " Status", Width: 9},
		{Title: theme.Repo + " Path", Width: max(16, width-31)},
	})
	m.sourceTable.SetWidth(width)
	m.sourceTable.SetHeight(max(3, min(5, len(m.meta.Discovery.Sources)+1)))
	m.syncSourceTable()
}

func (m *Model) syncSourceTable() {
	rows := make([]table.Row, 0, len(m.meta.Discovery.Sources))
	for _, source := range m.meta.Discovery.Sources {
		status := "missing"
		if source.Exists {
			status = "present"
		}
		rows = append(
			rows,
			table.Row{source.Tool, source.Kind, status, uiutil.ShortenPath(source.Path)},
		)
	}
	m.sourceTable.SetRows(rows)
	m.sourceTable.SetCursor(0)
}
