package tui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yantonov/crtokt/src/config"
	"github.com/yantonov/crtokt/src/models"
	"github.com/yantonov/crtokt/src/tui/components"
)

// field indices — order matches tab navigation.
const (
	fieldEnv       = 0
	fieldSeverity  = 1
	fieldApp       = 2
	fieldAppCustom = 3 // only reachable when "custom..." is selected in appDD
	fieldTimeframe = 4
	fieldTraceID   = 5
	fieldQuery     = 6
	fieldCount     = 7
)

// customAppSentinel is the Option.Value used to signal "user will type a name".
const customAppSentinel = "__custom__"

// inputWidth matches the inner content width of the dropdown component.
const inputWidth = 50

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

// FilterScreen is both the initial standalone search-parameter form (shown after
// login) and the embedded filter panel used inside ResultsScreen.
type FilterScreen struct {
	cfg      config.Provider
	focusIdx int

	envDD          components.Dropdown
	severityDD     components.Dropdown
	appDD          components.Dropdown
	appCustomInput textinput.Model
	timeframeDD    components.Dropdown
	traceInput     textinput.Model
	queryInput     textinput.Model
}

// NewFilterScreen constructs a FilterScreen populated from cfg.
func NewFilterScreen(cfg config.Provider) FilterScreen {
	// Environments (sorted for determinism).
	envKeys := make([]string, 0, len(cfg.Environments()))
	for k := range cfg.Environments() {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	envOpts := make([]components.Option, len(envKeys))
	for i, k := range envKeys {
		envOpts[i] = components.Option{Label: k, Value: k}
	}

	// Severity: "all" first, then hardcoded syslog levels (Emergency→Debug).
	sevOpts := []components.Option{{Label: "all", Value: ""}}
	for _, pair := range models.AllSeverityOptions() {
		sevOpts = append(sevOpts, components.Option{Label: pair[0], Value: pair[1]})
	}

	// Applications: "all" first, then list from config, then "custom..." sentinel.
	appOpts := []components.Option{{Label: "all", Value: ""}}
	for _, a := range cfg.Applications() {
		appOpts = append(appOpts, components.Option{Label: a, Value: a})
	}
	appOpts = append(appOpts, components.Option{Label: "custom...", Value: customAppSentinel})

	// Timeframes: as-is from defaults.
	tfOpts := make([]components.Option, len(cfg.Timeframes()))
	for i, tf := range cfg.Timeframes() {
		tfOpts[i] = components.Option{Label: tf.Label, Value: tf.Value}
	}

	appCustomIn := textinput.New()
	appCustomIn.Placeholder = "application name"
	appCustomIn.CharLimit = 256
	appCustomIn.Width = inputWidth

	traceIn := textinput.New()
	traceIn.Placeholder = "optional"
	traceIn.CharLimit = 128
	traceIn.Width = inputWidth

	queryIn := textinput.New()
	queryIn.Placeholder = "Text"
	queryIn.CharLimit = 512
	queryIn.Width = inputWidth

	severityDD := components.New(sevOpts)
	severityDD.SetByValue("3") // Error by default

	timeframeDD := components.New(tfOpts)
	timeframeDD.SetByValue("3h")

	fs := FilterScreen{
		cfg:            cfg,
		envDD:          components.New(envOpts),
		severityDD:     severityDD,
		appDD:          components.New(appOpts),
		appCustomInput: appCustomIn,
		timeframeDD:    timeframeDD,
		traceInput:     traceIn,
		queryInput:     queryIn,
	}
	fs.syncFocus()
	return fs
}

// NewFilterScreenFromFilter creates a FilterScreen pre-populated with values from f.
func NewFilterScreenFromFilter(cfg config.Provider, f models.Filter) FilterScreen {
	fs := NewFilterScreen(cfg)
	fs.envDD.SetByValue(f.Environment)
	if f.Severity < 0 {
		fs.severityDD.SetByValue("") // "all"
	} else {
		fs.severityDD.SetByValue(strconv.Itoa(f.Severity))
	}
	if f.Application != "" {
		fs.appDD.SetByValue(f.Application)
		if fs.appDD.Selected().Value != f.Application {
			// not found in known apps → use custom
			fs.appDD.SetByValue(customAppSentinel)
			fs.appCustomInput.SetValue(f.Application)
		}
	}
	fs.timeframeDD.SetByValue(f.Timeframe)
	fs.traceInput.SetValue(f.TraceID)
	fs.queryInput.SetValue(f.Query)
	return fs
}

// ── standalone screen interface ───────────────────────────────────────────────

// Init satisfies the Screen interface.
func (f FilterScreen) Init() tea.Cmd {
	return textinput.Blink
}

// Update satisfies the Screen interface when used as a standalone screen.
// When embedded inside ResultsScreen, UpdateEmbedded is used instead.
func (f FilterScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	fs, cmd := f.update(msg)
	return fs, cmd
}

// View renders the full standalone filter form (title + fields + help line).
func (f FilterScreen) View() string {
	appField := f.appDD.View()
	if f.isCustomApp() {
		appField = lipgloss.JoinVertical(lipgloss.Left,
			appField,
			f.wrapInput(f.appCustomInput, f.focusIdx == fieldAppCustom),
		)
	}

	rows := []string{
		f.row("Environment", f.envDD.View()),
		f.row("Severity", f.severityDD.View()),
		f.row("Application", appField),
		f.row("Timeframe", f.timeframeDD.View()),
		f.row("Trace ID", f.wrapInput(f.traceInput, f.focusIdx == fieldTraceID)),
		f.row("Query", f.wrapInput(f.queryInput, f.focusIdx == fieldQuery)),
	}

	help := helpStyle.Render("ctrl+s search  ·  tab next  ·  shift+tab prev  ·  ctrl+c quit")

	return lipgloss.NewStyle().Padding(1, 2).Render(
		strings.Join([]string{
			titleStyle.Render("crtokt — Log Viewer"),
			"",
			strings.Join(rows, "\n"),
			"",
			help,
		}, "\n"),
	)
}

// ── embedded panel interface ──────────────────────────────────────────────────

// UpdateEmbedded handles key and tick messages routed from ResultsScreen.
func (f FilterScreen) UpdateEmbedded(msg tea.Msg) (FilterScreen, tea.Cmd) {
	return f.update(msg)
}

// ViewEmbedded renders the filter fields without title, help or outer padding.
func (f FilterScreen) ViewEmbedded() string {
	appField := f.appDD.View()
	if f.isCustomApp() {
		appField = lipgloss.JoinVertical(lipgloss.Left,
			appField,
			f.wrapInput(f.appCustomInput, f.focusIdx == fieldAppCustom),
		)
	}
	rows := []string{
		f.row("Environment", f.envDD.View()),
		f.row("Severity", f.severityDD.View()),
		f.row("Application", appField),
		f.row("Timeframe", f.timeframeDD.View()),
		f.row("Trace ID", f.wrapInput(f.traceInput, f.focusIdx == fieldTraceID)),
		f.row("Query", f.wrapInput(f.queryInput, f.focusIdx == fieldQuery)),
	}
	return strings.Join(rows, "\n")
}

// ── focus helpers ─────────────────────────────────────────────────────────────

// AtFirstField reports whether focus is on the first reachable field.
func (f FilterScreen) AtFirstField() bool { return f.focusIdx == fieldEnv }

// AtLastField reports whether focus is on the last reachable field.
func (f FilterScreen) AtLastField() bool { return f.focusIdx == fieldQuery }

// FocusFirst moves focus to the first field.
func (f *FilterScreen) FocusFirst() { f.focusIdx = fieldEnv; f.syncFocus() }

// FocusLast moves focus to the last field.
func (f *FilterScreen) FocusLast() { f.focusIdx = fieldQuery; f.syncFocus() }

// FocusNone blurs all fields (used when the results table takes focus).
func (f *FilterScreen) FocusNone() { f.focusIdx = -1; f.syncFocus() }

// IsAnyDropdownExpanded reports whether any dropdown is currently open.
func (f FilterScreen) IsAnyDropdownExpanded() bool {
	return f.envDD.IsExpanded() || f.severityDD.IsExpanded() ||
		f.appDD.IsExpanded() || f.timeframeDD.IsExpanded()
}

// ── shared update logic ───────────────────────────────────────────────────────

func (f FilterScreen) update(msg tea.Msg) (FilterScreen, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		return f, nil
	}

	key, isKey := msg.(tea.KeyMsg)

	if !isKey {
		return f.updateActiveInput(msg)
	}

	if key.String() == "ctrl+s" {
		return f, f.triggerSearch()
	}

	switch key.String() {
	case "tab":
		f.collapseActiveDrop()
		f.focusIdx = f.nextField()
		f.syncFocus()
		return f, nil
	case "shift+tab":
		f.collapseActiveDrop()
		f.focusIdx = f.prevField()
		f.syncFocus()
		return f, nil
	}

	if f.focusIdx == fieldAppCustom && !f.isCustomApp() {
		f.focusIdx = fieldApp
		f.syncFocus()
	}

	return f.routeKey(key)
}

// ── private helpers ───────────────────────────────────────────────────────────

func (f *FilterScreen) syncFocus() {
	f.envDD.SetFocused(f.focusIdx == fieldEnv)
	f.severityDD.SetFocused(f.focusIdx == fieldSeverity)
	f.appDD.SetFocused(f.focusIdx == fieldApp)
	f.timeframeDD.SetFocused(f.focusIdx == fieldTimeframe)

	if f.focusIdx == fieldAppCustom {
		f.appCustomInput.Focus()
	} else {
		f.appCustomInput.Blur()
	}
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

func (f FilterScreen) isCustomApp() bool {
	return f.appDD.Selected().Value == customAppSentinel
}

func (f *FilterScreen) nextField() int {
	next := (f.focusIdx + 1) % fieldCount
	if next == fieldAppCustom && !f.isCustomApp() {
		next = (next + 1) % fieldCount
	}
	return next
}

func (f *FilterScreen) prevField() int {
	prev := (f.focusIdx - 1 + fieldCount) % fieldCount
	if prev == fieldAppCustom && !f.isCustomApp() {
		prev = (prev - 1 + fieldCount) % fieldCount
	}
	return prev
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
	case fieldAppCustom:
		var cmd tea.Cmd
		f.appCustomInput, cmd = f.appCustomInput.Update(key)
		return f, cmd
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
	case fieldAppCustom:
		var cmd tea.Cmd
		f.appCustomInput, cmd = f.appCustomInput.Update(msg)
		return f, cmd
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
	severity := models.SeverityAll
	if v := f.severityDD.Selected().Value; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			severity = n
		}
	}
	app := f.appDD.Selected().Value
	if app == customAppSentinel {
		app = strings.TrimSpace(f.appCustomInput.Value())
	}
	filter := models.Filter{
		Environment: f.envDD.Selected().Value,
		Severity:    severity,
		Application: app,
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
