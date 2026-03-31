package screens

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"github.com/criteo/klt/src/tui/components"
)

// SearchStartedMsg is sent when the user triggers a search from the filter screen.
type SearchStartedMsg struct {
	Filter models.Filter
}

// field indices — order matches tab navigation.
const (
	fieldEnv       = 0
	fieldSeverity  = 1
	fieldApp       = 2
	fieldTimeframe = 3
	fieldTraceID   = 4
	fieldQuery     = 5
	fieldCount     = 6
)

// inputWidth matches the inner content width of the dropdown component.
const inputWidth = 30

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	labelStyle = lipgloss.NewStyle().
			Width(14).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("#D1D5DB"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	inputFocused = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			PaddingLeft(1).PaddingRight(1).
			Width(inputWidth + 2)

	inputBlurred = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6B7280")).
			PaddingLeft(1).PaddingRight(1).
			Width(inputWidth + 2)
)

// FilterScreen is the initial search-parameter form.
type FilterScreen struct {
	cfg      *config.Config
	width    int
	height   int
	focusIdx int

	envDD       components.Dropdown
	severityDD  components.Dropdown
	appDD       components.Dropdown
	timeframeDD components.Dropdown
	traceInput  textinput.Model
	queryInput  textinput.Model
}

// NewFilterScreen constructs a FilterScreen populated from cfg.
func NewFilterScreen(cfg *config.Config) FilterScreen {
	// Environments (sorted for determinism).
	envKeys := make([]string, 0, len(cfg.Environments))
	for k := range cfg.Environments {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	envOpts := make([]components.Option, len(envKeys))
	for i, k := range envKeys {
		envOpts[i] = components.Option{Label: k, Value: k}
	}

	// Severity: "all" first, then levels from config.
	sevOpts := []components.Option{{Label: "all", Value: ""}}
	for _, s := range cfg.SeverityLevels {
		sevOpts = append(sevOpts, components.Option{Label: s, Value: s})
	}

	// Applications: "all" first, then list from config.
	appOpts := []components.Option{{Label: "all", Value: ""}}
	for _, a := range cfg.Applications {
		appOpts = append(appOpts, components.Option{Label: a, Value: a})
	}

	// Timeframes: as-is from config.
	tfOpts := make([]components.Option, len(cfg.Timeframes))
	for i, tf := range cfg.Timeframes {
		tfOpts[i] = components.Option{Label: tf.Label, Value: tf.Value}
	}

	traceIn := textinput.New()
	traceIn.Placeholder = "optional"
	traceIn.CharLimit = 128
	traceIn.Width = inputWidth

	queryIn := textinput.New()
	queryIn.Placeholder = "Lucene / KQL"
	queryIn.CharLimit = 512
	queryIn.Width = inputWidth

	fs := FilterScreen{
		cfg:         cfg,
		envDD:       components.New(envOpts),
		severityDD:  components.New(sevOpts),
		appDD:       components.New(appOpts),
		timeframeDD: components.New(tfOpts),
		traceInput:  traceIn,
		queryInput:  queryIn,
	}
	fs.syncFocus()
	return fs
}

// Init satisfies the screen interface.
func (f FilterScreen) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all messages for the filter screen.
func (f FilterScreen) Update(msg tea.Msg) (FilterScreen, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		f.width = ws.Width
		f.height = ws.Height
		return f, nil
	}

	key, isKey := msg.(tea.KeyMsg)

	if !isKey {
		// Forward to the active textinput for cursor-blink ticks etc.
		return f.updateActiveInput(msg)
	}

	// Global shortcut — trigger search.
	if key.String() == "ctrl+s" {
		return f, f.triggerSearch()
	}

	// Tab / Shift+Tab always cycle focus (collapsing any open dropdown first).
	switch key.String() {
	case "tab":
		f.collapseActiveDrop()
		f.focusIdx = (f.focusIdx + 1) % fieldCount
		f.syncFocus()
		return f, nil
	case "shift+tab":
		f.collapseActiveDrop()
		f.focusIdx = (f.focusIdx - 1 + fieldCount) % fieldCount
		f.syncFocus()
		return f, nil
	}

	// Route the key to the active field.
	return f.routeKey(key)
}

// View renders the filter form.
func (f FilterScreen) View() string {
	rows := []string{
		f.row("Environment", f.envDD.View()),
		f.row("Severity", f.severityDD.View()),
		f.row("Application", f.appDD.View()),
		f.row("Timeframe", f.timeframeDD.View()),
		f.row("Trace ID", f.wrapInput(f.traceInput, f.focusIdx == fieldTraceID)),
		f.row("Query", f.wrapInput(f.queryInput, f.focusIdx == fieldQuery)),
	}

	help := helpStyle.Render("ctrl+s search  ·  tab next  ·  shift+tab prev  ·  ctrl+c quit")

	return lipgloss.NewStyle().Padding(1, 2).Render(
		strings.Join([]string{
			titleStyle.Render("klt — Log Viewer"),
			"",
			strings.Join(rows, "\n"),
			"",
			help,
		}, "\n"),
	)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (f *FilterScreen) syncFocus() {
	f.envDD.SetFocused(f.focusIdx == fieldEnv)
	f.severityDD.SetFocused(f.focusIdx == fieldSeverity)
	f.appDD.SetFocused(f.focusIdx == fieldApp)
	f.timeframeDD.SetFocused(f.focusIdx == fieldTimeframe)

	if f.focusIdx == fieldTraceID {
		f.traceInput.Focus()
	} else {
		f.traceInput.Blur()
	}
	if f.focusIdx == fieldQuery {
		f.queryInput.Focus()
	} else {
		f.queryInput.Blur()
	}
}

func (f *FilterScreen) collapseActiveDrop() {
	switch f.focusIdx {
	case fieldEnv:
		f.envDD.Collapse()
	case fieldSeverity:
		f.severityDD.Collapse()
	case fieldApp:
		f.appDD.Collapse()
	case fieldTimeframe:
		f.timeframeDD.Collapse()
	}
}

func (f FilterScreen) routeKey(key tea.KeyMsg) (FilterScreen, tea.Cmd) {
	switch f.focusIdx {
	case fieldEnv:
		f.envDD, _ = f.envDD.Update(key)
	case fieldSeverity:
		f.severityDD, _ = f.severityDD.Update(key)
	case fieldApp:
		f.appDD, _ = f.appDD.Update(key)
	case fieldTimeframe:
		f.timeframeDD, _ = f.timeframeDD.Update(key)
	case fieldTraceID:
		var cmd tea.Cmd
		f.traceInput, cmd = f.traceInput.Update(key)
		return f, cmd
	case fieldQuery:
		var cmd tea.Cmd
		f.queryInput, cmd = f.queryInput.Update(key)
		return f, cmd
	}
	return f, nil
}

func (f FilterScreen) updateActiveInput(msg tea.Msg) (FilterScreen, tea.Cmd) {
	switch f.focusIdx {
	case fieldTraceID:
		var cmd tea.Cmd
		f.traceInput, cmd = f.traceInput.Update(msg)
		return f, cmd
	case fieldQuery:
		var cmd tea.Cmd
		f.queryInput, cmd = f.queryInput.Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f FilterScreen) triggerSearch() tea.Cmd {
	filter := models.Filter{
		Environment: f.envDD.Selected().Value,
		Severity:    f.severityDD.Selected().Value,
		Application: f.appDD.Selected().Value,
		Timeframe:   f.timeframeDD.Selected().Value,
		TraceID:     strings.TrimSpace(f.traceInput.Value()),
		Query:       strings.TrimSpace(f.queryInput.Value()),
	}
	return func() tea.Msg { return SearchStartedMsg{Filter: filter} }
}

func (f FilterScreen) row(label, field string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render(label)+"  ",
		field,
	)
}

func (f FilterScreen) wrapInput(ti textinput.Model, focused bool) string {
	if focused {
		return inputFocused.Render(ti.View())
	}
	return inputBlurred.Render(ti.View())
}
