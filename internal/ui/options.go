package ui

import (
	"slices"

	"github.com/samber/lo"

	"github.com/adin/ai-dash/internal/session"
)

func toolOptions(sessions []session.Session) []string {
	return append(
		[]string{""},
		uniqueSortedValues(sessions, func(s session.Session) string { return s.Tool })...)
}

func projectOptions(sessions []session.Session) []string {
	return append(
		[]string{""},
		uniqueSortedValues(sessions, func(s session.Session) string { return s.Project })...)
}

func uniqueSortedValues(sessions []session.Session, pick func(session.Session) string) []string {
	values := lo.Uniq(lo.FilterMap(sessions, func(s session.Session, _ int) (string, bool) {
		v := pick(s)
		return v, v != ""
	}))
	slices.Sort(values)
	return values
}

func nextSortField(current session.SortField) session.SortField {
	fields := []session.SortField{
		session.SortUpdated,
		session.SortTool,
		session.SortStatus,
		session.SortProject,
		session.SortSummary,
	}
	for i, field := range fields {
		if field == current {
			return fields[(i+1)%len(fields)]
		}
	}
	return session.SortUpdated
}

func prevSortField(current session.SortField) session.SortField {
	fields := []session.SortField{
		session.SortUpdated,
		session.SortTool,
		session.SortStatus,
		session.SortProject,
		session.SortSummary,
	}
	for i, field := range fields {
		if field == current {
			return fields[(i-1+len(fields))%len(fields)]
		}
	}
	return session.SortUpdated
}
