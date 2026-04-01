package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"github.com/criteo/klt/src/opensearch"
)

// App is the root Bubble Tea model. It owns the screen-routing state machine
// and delegates Update/View to the active Screen.
type App struct {
	cfg    config.Provider
	client opensearch.Searcher

	screen   Screen
	showHelp bool
	width    int
	height   int

	// loading state while a search or login is in-flight
	loading       bool
	loadingFilter models.Filter

	// stored so the results screen can be rebuilt when navigating back from detail
	lastResult models.CombinedResult
	lastFilter models.Filter

	spinner spinner.Model
}

// New constructs the root App model.
func New(cfg config.Provider, client opensearch.Searcher) App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	return App{
		cfg:     cfg,
		client:  client,
		screen:  NewLoginScreen(),
		spinner: s,
	}
}

// Init satisfies tea.Model.
func (a App) Init() tea.Cmd {
	return a.screen.Init()
}

// Update is the root message dispatcher.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		model, cmd := a.screen.Update(msg)
		a.screen = model
		return a, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "?":
			if _, isLogin := a.screen.(LoginScreen); !isLogin {
				a.showHelp = !a.showHelp
			}
			return a, nil
		case "esc":
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if _, isFilter := a.screen.(FilterScreen); isFilter {
				return a, tea.Quit
			}
		}
		if a.showHelp {
			// Any other key closes the help overlay.
			a.showHelp = false
			return a, nil
		}

	// User submitted the login form.
	case LoginSubmitMsg:
		a.loading = true
		return a, tea.Batch(a.doLogin(msg.Username, msg.Password), a.spinner.Tick)

	// Login succeeded.
	case LoginDoneMsg:
		a.loading = false
		a.screen = NewFilterScreen(a.cfg)
		return a, a.screen.Init()

	// Login failed — delegate to the login screen so it can display the error.
	case loginErrMsg:
		a.loading = false
		model, cmd := a.screen.Update(msg)
		a.screen = model
		return a, cmd

	// Search was triggered from FilterScreen.
	case SearchStartedMsg:
		a.loading = true
		a.loadingFilter = msg.Filter
		return a, tea.Batch(a.doSearch(msg.Filter), a.spinner.Tick)

	// Parallel search completed.
	case SearchDoneMsg:
		a.loading = false
		a.lastResult = msg.Result
		a.lastFilter = msg.Filter
		a.screen = NewResultsScreen(msg.Result, msg.Filter, a.width, a.height)
		return a, nil

	// Spinner tick while loading.
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	// User wants to refine the search.
	case BackToFilterMsg:
		a.screen = NewFilterScreen(a.cfg)
		return a, nil

	// User wants to re-run the same search.
	case RefreshMsg:
		a.loading = true
		a.loadingFilter = msg.Filter
		return a, tea.Batch(a.doSearch(msg.Filter), a.spinner.Tick)

	// User selected a log entry.
	case OpenDetailMsg:
		kibanaBase := a.cfg.KibanaURL(msg.Entry.DataCenter, msg.Entry.Environment)
		a.screen = NewDetailScreen(msg.Entry, kibanaBase, a.width, a.height)
		return a, nil

	// User navigates back from detail to results.
	case BackToResultsMsg:
		a.screen = NewResultsScreen(a.lastResult, a.lastFilter, a.width, a.height)
		return a, nil
	}

	if a.showHelp {
		return a, nil
	}

	// Delegate to the active screen.
	model, cmd := a.screen.Update(msg)
	a.screen = model
	return a, cmd
}

// View renders the active screen or the help/loading overlay.
func (a App) View() string {
	if a.showHelp {
		return helpView(a.width, a.height)
	}
	if a.loading {
		if _, isLogin := a.screen.(LoginScreen); isLogin {
			return a.screen.View() + "\n\n  " + a.spinner.View() + " Authenticating…"
		}
		return a.loadingView()
	}
	return a.screen.View()
}

var (
	loadingBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 1)

	loadingHighlight = lipgloss.NewStyle().
				Background(lipgloss.Color("#111827")).
				Foreground(lipgloss.Color("#C4B5FD"))
)

func (a App) loadingView() string {
	f := a.loadingFilter
	hi := loadingHighlight.Render

	app := "all"
	if f.Application != "" {
		app = f.Application
	}
	sev := "all"
	if f.Severity >= 0 {
		sev = models.SeverityLabel(f.Severity)
	}

	summary := fmt.Sprintf("env:%s  app:%s  severity:%s  timeframe:%s",
		hi(f.Environment), hi(app), hi(sev), hi(f.Timeframe))
	if f.TraceID != "" {
		summary += fmt.Sprintf("  trace:%s", hi(f.TraceID))
	}

	bar := loadingBarStyle.Width(a.width).Render(summary)
	msg := "\n  " + a.spinner.View() + " Searching…"
	return bar + msg
}

// doLogin performs the authentication request as a tea.Cmd.
func (a App) doLogin(username, password string) tea.Cmd {
	cfg := a.cfg
	client := a.client
	return func() tea.Msg {
		// Use the first available DC to authenticate.
		var kibanaURL string
		for e, ecfg := range cfg.Environments() {
			if len(ecfg.DataCenters) > 0 {
				kibanaURL = cfg.KibanaURL(ecfg.DataCenters[0], e)
				break
			}
		}
		if kibanaURL == "" {
			return loginErrMsg{err: fmt.Errorf("no environments configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Login(ctx, kibanaURL, username, password); err != nil {
			return loginErrMsg{err: err}
		}
		return LoginDoneMsg{}
	}
}

// doSearch launches the parallel OpenSearch fanout as a tea.Cmd.
func (a App) doSearch(filter models.Filter) tea.Cmd {
	cfg := a.cfg
	client := a.client
	return func() tea.Msg {
		result := client.SearchAll(context.Background(), cfg, filter)
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
