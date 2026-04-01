package screens

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/criteo/klt/src/export"
	"github.com/criteo/klt/src/models"
)

// BackToFilterMsg is sent when the user wants to refine the search.
type BackToFilterMsg struct{}

// RefreshMsg is sent when the user wants to re-run the same search.
type RefreshMsg struct {
	Filter models.Filter
}

// OpenDetailMsg is sent when the user selects a log entry.
type OpenDetailMsg struct {
	Entry models.LogEntry
}

var (
	resultsStatusBar = lipgloss.NewStyle().
				Background(lipgloss.Color("#1F2937")).
				Foreground(lipgloss.Color("#F9FAFB")).
				Padding(0, 1)

	resultsDCOK  = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	resultsDCErr = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))

	resultsFilterInput = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				PaddingLeft(1).PaddingRight(1)

	resultsFilterBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 1)

	resultsFilterHighlight = lipgloss.NewStyle().
				Background(lipgloss.Color("#111827")).
				Foreground(lipgloss.Color("#C4B5FD"))
)

// ResultsScreen displays the merged search results in a table.
type ResultsScreen struct {
	result  models.CombinedResult
	filter  models.Filter
	allDCs  []string // sorted list of all DCs that were searched
	ready   bool     // true after NewResultsScreen has been called

	tbl         table.Model
	filterInput textinput.Model
	filtering   bool
	filtered    []models.LogEntry // currently visible subset (nil = show all)
	notice      string            // transient feedback shown in the status bar

	width  int
	height int
}

// NewResultsScreen constructs a ResultsScreen from the combined search result.
func NewResultsScreen(result models.CombinedResult, filter models.Filter, width, height int) ResultsScreen {
	dcSet := make(map[string]struct{})
	for _, e := range result.Entries {
		dcSet[e.DataCenter] = struct{}{}
	}
	for dc := range result.DCErrors {
		dcSet[dc] = struct{}{}
	}
	allDCs := make([]string, 0, len(dcSet))
	for dc := range dcSet {
		allDCs = append(allDCs, dc)
	}
	sort.Strings(allDCs)

	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 128

	rs := ResultsScreen{
		result:      result,
		filter:      filter,
		allDCs:      allDCs,
		filterInput: fi,
		width:       width,
		height:      height,
		ready:       true,
	}
	rs.tbl = rs.buildTable(result.Entries)
	return rs
}

// Init satisfies the screen interface.
func (rs ResultsScreen) Init() tea.Cmd { return nil }

// Update handles all messages for the results screen.
func (rs ResultsScreen) Update(msg tea.Msg) (ResultsScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rs.width = msg.Width
		rs.height = msg.Height
		rs.tbl = rs.buildTable(rs.visibleEntries())
		return rs, nil

	case tea.KeyMsg:
		if rs.filtering {
			return rs.handleFilterKey(msg)
		}
		return rs.handleKey(msg)
	}
	return rs, nil
}

// View renders the results table and status bar.
func (rs ResultsScreen) View() string {
	if !rs.ready {
		return "  Searching…"
	}

	var parts []string

	parts = append(parts, rs.filterSummary())
	if rs.filtering {
		parts = append(parts, "  / "+resultsFilterInput.Render(rs.filterInput.View()))
	}
	parts = append(parts, rs.tbl.View())
	parts = append(parts, rs.statusBar())

	return strings.Join(parts, "\n")
}

// ── key handling ─────────────────────────────────────────────────────────────

func (rs ResultsScreen) handleKey(msg tea.KeyMsg) (ResultsScreen, tea.Cmd) {
	rs.notice = "" // clear previous notice on any keypress

	switch msg.String() {
	case "ctrl+r":
		f := rs.filter
		return rs, func() tea.Msg { return RefreshMsg{Filter: f} }

	case "r":
		return rs, func() tea.Msg { return BackToFilterMsg{} }

	case "enter":
		entries := rs.visibleEntries()
		cur := rs.tbl.Cursor()
		if cur >= 0 && cur < len(entries) {
			entry := entries[cur]
			return rs, func() tea.Msg { return OpenDetailMsg{Entry: entry} }
		}

	case "e":
		entries := rs.visibleEntries()
		path, err := export.WriteNDJSON(entries)
		if err != nil {
			rs.notice = "export failed: " + err.Error()
		} else {
			rs.notice = fmt.Sprintf("exported %d results → %s", len(entries), path)
		}
		return rs, nil

	case "c":
		entries := rs.visibleEntries()
		cur := rs.tbl.Cursor()
		if cur >= 0 && cur < len(entries) {
			if err := export.CopyJSON(entries[cur].RawJSON); err != nil {
				rs.notice = "copy failed: " + err.Error()
			} else {
				rs.notice = "copied to clipboard"
			}
		}
		return rs, nil

	case "/":
		rs.filtering = true
		rs.filterInput.Focus()
		return rs, textinput.Blink

	case "esc":
		return rs, func() tea.Msg { return BackToFilterMsg{} }
	}

	var cmd tea.Cmd
	rs.tbl, cmd = rs.tbl.Update(msg)
	return rs, cmd
}

func (rs ResultsScreen) handleFilterKey(msg tea.KeyMsg) (ResultsScreen, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		rs.filtering = false
		rs.filterInput.Blur()
		return rs, nil
	}

	var cmd tea.Cmd
	rs.filterInput, cmd = rs.filterInput.Update(msg)

	query := strings.ToLower(rs.filterInput.Value())
	if query == "" {
		rs.filtered = nil
	} else {
		rs.filtered = nil
		for _, e := range rs.result.Entries {
			if strings.Contains(strings.ToLower(e.Message), query) ||
				strings.Contains(strings.ToLower(e.Application), query) ||
				strings.Contains(strings.ToLower(e.TraceID), query) {
				rs.filtered = append(rs.filtered, e)
			}
		}
	}
	rs.tbl = rs.buildTable(rs.visibleEntries())
	return rs, cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (rs ResultsScreen) visibleEntries() []models.LogEntry {
	if rs.filtered != nil {
		return rs.filtered
	}
	return rs.result.Entries
}

func (rs ResultsScreen) buildTable(entries []models.LogEntry) table.Model {
	// Leave room for filter summary (1), status bar (1), optional filter input (1), plus table header.
	tableHeight := rs.height - 4
	if tableHeight < 3 {
		tableHeight = 3
	}

	const (
		tsWidth  = 20
		sevWidth = 14
		dcWidth  = 8
		appWidth = 20
		gaps     = 10
	)
	msgWidth := rs.width - tsWidth - sevWidth - dcWidth - appWidth - gaps
	if msgWidth < 20 {
		msgWidth = 20
	}

	cols := []table.Column{
		{Title: "Timestamp", Width: tsWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "DC", Width: dcWidth},
		{Title: "Application", Width: appWidth},
		{Title: "Message", Width: msgWidth},
	}

	rows := make([]table.Row, len(entries))
	for i, e := range entries {
		rows[i] = table.Row{
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Severity,
			e.DataCenter,
			e.Application,
			truncate(e.Message, msgWidth),
		}
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6B7280")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Bold(true)

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithStyles(s),
	)
	return t
}

func (rs ResultsScreen) statusBar() string {
	var dcParts []string
	for _, dc := range rs.allDCs {
		if _, hasErr := rs.result.DCErrors[dc]; hasErr {
			dcParts = append(dcParts, resultsDCErr.Render("● "+dc))
		} else {
			dcParts = append(dcParts, resultsDCOK.Render("● "+dc))
		}
	}
	dcSection := strings.Join(dcParts, " ")

	entries := rs.visibleEntries()
	count := fmt.Sprintf("%d results", len(entries))
	if rs.filtered != nil {
		count = fmt.Sprintf("%d / %d results", len(entries), len(rs.result.Entries))
	}

	var right string
	if rs.notice != "" {
		right = rs.notice
	} else {
		right = "↑↓/jk navigate · enter detail · / filter · r refine · ctrl+r refresh · e export · c copy"
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left,
		dcSection+"  ",
		count+"    ",
		right,
	)
	return resultsStatusBar.Width(rs.width).Render(content)
}

func (rs ResultsScreen) filterSummary() string {
	f := rs.filter
	hi := resultsFilterHighlight.Render

	app := "all"
	if f.Application != "" {
		app = f.Application
	}

	sev := "all"
	if f.Severity >= 0 {
		sev = models.SeverityLabel(f.Severity)
	}

	line := fmt.Sprintf("env:%s  app:%s  severity:%s  timeframe:%s",
		hi(f.Environment), hi(app), hi(sev), hi(f.Timeframe))
	if f.TraceID != "" {
		line += fmt.Sprintf("  trace:%s", hi(f.TraceID))
	}
	return resultsFilterBar.Width(rs.width).Render(line)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}
