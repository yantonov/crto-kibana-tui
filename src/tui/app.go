package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

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

	active screen
	width  int
	height int

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

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	// Search was triggered from FilterScreen.
	case screens.SearchStartedMsg:
		a.active = screenResults
		return a, a.doSearch(msg.Filter)

	// Parallel search completed.
	case SearchDoneMsg:
		a.result = msg.Result
		a.selectedIdx = 0
		a.resultsScreen = screens.NewResultsScreen(msg.Result, a.width, a.height)
		a.active = screenResults
		return a, nil

	// User wants to refine the search.
	case screens.BackToFilterMsg:
		a.active = screenFilter
		return a, nil

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

// View renders the active screen.
func (a App) View() string {
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
		return SearchDoneMsg{Result: result}
	}
}
