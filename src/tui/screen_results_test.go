package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yantonov/crtokt/src/models"
)

func resultsScreen(t *testing.T) ResultsScreen {
	t.Helper()
	result := models.CombinedResult{
		Entries: []models.LogEntry{
			{Timestamp: time.Unix(0, 0).UTC(), Severity: "E", DataCenter: "dc1", Application: "app", Message: "boom"},
			{Timestamp: time.Unix(1, 0).UTC(), Severity: "I", DataCenter: "dc1", Application: "app", Message: "fine"},
		},
	}
	filter := models.Filter{Environment: "preprod", Severity: -1, Timeframe: "1h"}
	return NewResultsScreen(result, filter, stubConfig{}, 120, 40)
}

func press(t *testing.T, rs ResultsScreen, msg tea.KeyMsg) ResultsScreen {
	t.Helper()
	model, _ := rs.Update(msg)
	next, ok := model.(ResultsScreen)
	if !ok {
		t.Fatalf("Update returned %T, want ResultsScreen", model)
	}
	return next
}

var (
	escKey   = tea.KeyMsg{Type: tea.KeyEsc}
	tabKey   = tea.KeyMsg{Type: tea.KeyTab}
	slashKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	qKey     = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
)

func TestEscFromTheResultsTableFocusesTheFilterPanel(t *testing.T) {
	rs := resultsScreen(t)
	if rs.filterFocused {
		t.Fatal("a fresh results screen should start on the table")
	}

	rs = press(t, rs, escKey)

	if !rs.filterFocused {
		t.Fatal("esc should focus the filter panel")
	}
	if !rs.filterPanel.AtFirstField() {
		t.Fatal("esc should land on the first field, as tab does")
	}
}

func TestEscMatchesTabWhenLeavingTheResultsTable(t *testing.T) {
	viaEsc := press(t, resultsScreen(t), escKey)
	viaTab := press(t, resultsScreen(t), tabKey)

	if viaEsc.filterFocused != viaTab.filterFocused ||
		viaEsc.filterPanel.focusIdx != viaTab.filterPanel.focusIdx {
		t.Fatalf("esc focus (%v/%d) differs from tab focus (%v/%d)",
			viaEsc.filterFocused, viaEsc.filterPanel.focusIdx,
			viaTab.filterFocused, viaTab.filterPanel.focusIdx)
	}
}

func TestEscTogglesBackOutOfTheFilterPanel(t *testing.T) {
	rs := press(t, resultsScreen(t), escKey)

	rs = press(t, rs, escKey)

	if rs.filterFocused {
		t.Fatal("esc in the filter panel should return focus to the table")
	}
}

func TestEscClosesTheInlineFilterWithoutOpeningTheFilterPanel(t *testing.T) {
	rs := press(t, resultsScreen(t), slashKey)
	if !rs.filtering {
		t.Fatal("/ should start the inline filter")
	}

	rs = press(t, rs, escKey)

	if rs.filtering {
		t.Fatal("esc should close the inline filter")
	}
	if rs.filterFocused {
		t.Fatal("esc closing the inline filter should not also open the filter panel")
	}
}

// isQuit reports whether a command resolves to tea's quit message.
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, quit := cmd().(tea.QuitMsg)
	return quit
}

func TestQQuitsFromTheResultsTable(t *testing.T) {
	rs := resultsScreen(t)

	_, cmd := rs.Update(qKey)

	if !isQuit(cmd) {
		t.Fatal("q on the results table should quit")
	}
}

func TestQIsTypedIntoTheInlineFilterRatherThanQuitting(t *testing.T) {
	rs := press(t, resultsScreen(t), slashKey)

	next, cmd := rs.Update(qKey)
	rs = next.(ResultsScreen)

	if isQuit(cmd) {
		t.Fatal("q should not quit while the inline filter is open")
	}
	if got := rs.filterInput.Value(); got != "q" {
		t.Fatalf("inline filter holds %q, want %q", got, "q")
	}
}

func TestQIsTypedIntoTheFilterPanelQueryFieldRatherThanQuitting(t *testing.T) {
	rs := press(t, resultsScreen(t), escKey)
	rs.filterPanel.focusIdx = fieldQuery
	rs.filterPanel.syncFocus()

	next, cmd := rs.Update(qKey)
	rs = next.(ResultsScreen)

	if isQuit(cmd) {
		t.Fatal("q should not quit while a filter field has focus")
	}
	if got := rs.filterPanel.queryInput.Value(); got != "q" {
		t.Fatalf("query field holds %q, want %q", got, "q")
	}
}
