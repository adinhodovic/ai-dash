package ui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/adinhodovic/ai-dash/internal/session"
	uilayout "github.com/adinhodovic/ai-dash/internal/ui/layout"
	"github.com/adinhodovic/ai-dash/internal/ui/theme"
	uiutil "github.com/adinhodovic/ai-dash/internal/ui/util"
)

func detailPaneSectionHeights(termHeight int) (summary, detail int) {
	bodyH := uilayout.PaneBodyHeight(uilayout.BottomPaneHeight(termHeight))
	// Layout: summaryLabel(1) + summaryText(1) + divider(1) + detailTable
	summary = 2
	fixed := summary + 1 // summary section + divider/join line
	detail = max(3, bodyH-fixed)
	return summary, detail
}

func (m *Model) resizeDetailTable(filtered []session.Session) {
	sel := m.sessionTable.Cursor()
	if m.width == 0 || len(filtered) == 0 || sel < 0 || sel >= len(filtered) {
		m.detailTable.SetRows(nil)
		m.detailTable.UpdateViewport()
		return
	}
	detailW := m.width - m.width*70/100
	innerW := max(10, detailW-2-2*uilayout.PanePadding) // subtract pane border + padding
	_, detailH := detailPaneSectionHeights(m.height)
	keyW := 26
	valW := max(10, innerW-keyW-4) // subtract cell padding (1 each side × 2 cols)
	m.detailTable.SetColumns([]table.Column{
		{Title: "", Width: keyW},
		{Title: "", Width: valW},
	})
	m.detailTable.SetWidth(innerW)
	items := m.detailItems(filtered[sel])
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.title, uiutil.Truncate(item.desc, valW)})
	}
	m.detailTable.SetRows(rows)
	m.detailTable.SetHeight(detailH)
	m.detailTable.UpdateViewport()
}

func spacer() detailItem { return detailItem{"", ""} }

func (m Model) detailItems(s session.Session) []detailItem {
	// Identity
	items := []detailItem{
		{theme.Tool + " Tool", s.Tool},
		{theme.Project + " Project", uiutil.CleanProjectName(s.Project)},
		{theme.Active + " Status", uiutil.SessionStatusLabel(s)},
		{theme.Model + " Model", uiutil.ValueOrUnknown(s.Model)},
	}

	// Time
	items = append(items, spacer())
	items = append(items,
		detailItem{theme.Clock + " Active", uiutil.TimeAgo(uiutil.LastActive(s))},
		detailItem{theme.Clock + " Started", s.StartedAt.Format("2006-01-02 15:04:05")},
		detailItem{theme.Clock + " Ended", session.EndedLabel(s.EndedAt, s.Status)},
		detailItem{theme.Clock + " Duration", uiutil.DurationLabel(s)},
	)

	// Source
	var sourceItems []detailItem
	if s.Repo != "" {
		sourceItems = append(
			sourceItems,
			detailItem{theme.Repo + " Repo", uiutil.ShortenPath(s.Repo)},
		)
	}
	if s.Branch != "" {
		sourceItems = append(sourceItems, detailItem{theme.Branch + " Branch", s.Branch})
	}
	if len(sourceItems) > 0 {
		items = append(items, spacer())
		items = append(items, sourceItems...)
	}

	// Usage
	var usageItems []detailItem
	if s.TokensIn+s.TokensOut > 0 {
		usageItems = append(
			usageItems,
			detailItem{theme.Token + " Tokens", uiutil.FormatTokens(s.TokensIn, s.TokensOut)},
		)
	}
	if s.CostUSD > 0 {
		usageItems = append(
			usageItems,
			detailItem{theme.Cost + " Cost", uiutil.FormatCost(s.CostUSD)},
		)
	}
	if len(usageItems) > 0 {
		items = append(items, spacer())
		items = append(items, usageItems...)
	}

	// IDs
	items = append(items, spacer())
	if s.Slug != "" {
		items = append(items, detailItem{theme.Session + " Slug", s.Slug})
	}
	if s.ParentID != "" {
		items = append(items, detailItem{theme.Parent + " Parent", s.ParentID})
	}
	if len(s.Tags) > 0 {
		items = append(items, detailItem{theme.Tag + " Tags", strings.Join(s.Tags, ", ")})
	}
	items = append(items, detailItem{theme.ID + " ID", s.ID})

	// Metadata
	skip := map[string]bool{"model": true, "branch": true, "version": true}
	var metaItems []detailItem
	metaKeys := make([]string, 0, len(s.Meta))
	for k := range s.Meta {
		metaKeys = append(metaKeys, k)
	}
	slices.Sort(metaKeys)
	for _, k := range metaKeys {
		v := s.Meta[k]
		if !skip[k] {
			metaItems = append(metaItems, detailItem{"  " + uiutil.HumanizeKey(k), v})
		}
	}
	if len(metaItems) > 0 {
		items = append(items, spacer())
		items = append(items, detailItem{theme.Meta + " Metadata", ""})
		items = append(items, metaItems...)
	}
	return items
}
