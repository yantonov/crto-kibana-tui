package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"github.com/criteo/klt/src/opensearch"
	"github.com/criteo/klt/src/tui/screens"
)

// screen identifies which screen is currently active.
type screen int

const (
	screenFilter screen = iota
	screenResults
	screenDetail
)

// App is the root Bubble Tea model. It owns the screen-routing state machine
// and delegates Update/View to the active screen model.
type App struct {
	cfg    *config.Config
	client *opensearch.Client

	active   screen
	showHelp bool
	width    int
	height   int

	filterScreen  screens.FilterScreen
	resultsScreen screens.ResultsScreen
	detailScreen  screens.DetailScreen

	// populated after a search completes
	result      models.CombinedResult
	selectedIdx int
}

// New constructs the root App model.
func New(cfg *config.Config, client *opensearch.Client) App {
	return App{
		cfg:          cfg,
		client:       client,
		active:       screenFilter,
		filterScreen: screens.NewFilterScreen(cfg),
	}
}

// Init satisfies tea.Model.
func (a App) Init() tea.Cmd {
	return a.filterScreen.Init()
}

// Update is the root message dispatcher.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Update every screen so dimensions are correct on any transition.
		a.filterScreen, _ = a.filterScreen.Update(msg)
		a.resultsScreen, _ = a.resultsScreen.Update(msg)
		a.detailScreen, _ = a.detailScreen.Update(msg)
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "?":
			a.showHelp = !a.showHelp
			return a, nil
		case "esc":
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if a.active == screenFilter {
				return a, tea.Quit
			}
		}
		if a.showHelp {
			// All other keys close the help overlay.
			a.showHelp = false
			return a, nil
		}

	// Search was triggered from FilterScreen.
	case screens.SearchStartedMsg:
		a.active = screenResults
		return a, a.doSearch(msg.Filter)

	// Parallel search completed.
	case SearchDoneMsg:
		a.result = msg.Result
		a.selectedIdx = 0
		a.resultsScreen = screens.NewResultsScreen(msg.Result, msg.Filter, a.width, a.height)
		a.active = screenResults
		return a, nil

	// User wants to refine the search.
	case screens.BackToFilterMsg:
		a.active = screenFilter
		return a, nil

	// User wants to re-run the same search.
	case screens.RefreshMsg:
		a.active = screenFilter
		return a, a.doSearch(msg.Filter)

	// User selected a log entry.
	case screens.OpenDetailMsg:
		kibanaBase := a.cfg.KibanaURL(msg.Entry.DataCenter, msg.Entry.Environment)
		a.detailScreen = screens.NewDetailScreen(msg.Entry, kibanaBase, a.width, a.height)
		a.active = screenDetail
		return a, nil

	// User navigates back from detail to results.
	case screens.BackToResultsMsg:
		a.active = screenResults
		return a, nil
	}

	if a.showHelp {
		return a, nil
	}

	// Delegate to the active screen.
	switch a.active {
	case screenFilter:
		var cmd tea.Cmd
		a.filterScreen, cmd = a.filterScreen.Update(msg)
		return a, cmd
	case screenResults:
		var cmd tea.Cmd
		a.resultsScreen, cmd = a.resultsScreen.Update(msg)
		return a, cmd
	case screenDetail:
		var cmd tea.Cmd
		a.detailScreen, cmd = a.detailScreen.Update(msg)
		return a, cmd
	}

	return a, nil
}

// View renders the active screen or the help overlay.
func (a App) View() string {
	if a.showHelp {
		return helpView(a.width, a.height)
	}
	switch a.active {
	case screenFilter:
		return a.filterScreen.View()
	case screenResults:
		return a.resultsScreen.View()
	case screenDetail:
		return a.detailScreen.View()
	default:
		return ""
	}
}

// doSearch launches the parallel OpenSearch fanout as a tea.Cmd.
func (a App) doSearch(filter models.Filter) tea.Cmd {
	cfg := a.cfg
	client := a.client
	return func() tea.Msg {
		result := opensearch.SearchAll(context.Background(), filter, cfg, client)
		return SearchDoneMsg{Result: result, Filter: filter}
	}
}

// ── help overlay ──────────────────────────────────────────────────────────────

var (
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#D1D5DB"))

	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 3)
)

func helpView(width, height int) string {
	lines := []string{
		helpTitleStyle.Render("klt — Key Bindings"),
		"",
		helpSectionStyle.Render("Filter Screen"),
		"  ctrl+s        search",
		"  tab            next field",
		"  shift+tab     prev field",
		"  enter          confirm dropdown",
		"  esc             quit",
		"",
		helpSectionStyle.Render("Results Screen"),
		"  ↑/↓  j/k       navigate rows",
		"  enter           open detail",
		"  r                back to filter",
		"  ctrl+r          refresh (re-run search)",
		"  /                inline filter",
		"  esc             back to filter",
		"  e                export to NDJSON file",
		"  c                copy selected row JSON",
		"",
		helpSectionStyle.Render("Detail Screen"),
		"  ↑/↓  j/k       scroll",
		"  r                toggle raw / formatted",
		"  c                copy entry JSON",
		"  o                open in Kibana",
		"  esc / b          back to results",
		"",
		helpSectionStyle.Render("Global"),
		"  ?                toggle this help",
		"  ctrl+c           quit",
	}
	box := helpBoxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
