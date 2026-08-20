package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yantonov/crtokt/src/config"
	"github.com/yantonov/crtokt/src/credentials"
	"github.com/yantonov/crtokt/src/models"
	"github.com/yantonov/crtokt/src/opensearch"
)

// App is the root Bubble Tea model. It owns the screen-routing state machine
// and delegates Update/View to the active Screen.
type App struct {
	cfg    config.Provider
	client opensearch.Searcher
	store  credentials.Store

	// stored is what the keychain held at startup; pending is the pair the
	// in-flight login is using. They differ once the user types new
	// credentials, which is exactly when the keychain needs rewriting.
	stored  credentials.Credentials
	pending credentials.Credentials

	screen   Screen
	showHelp bool
	width    int
	height   int

	// loading state while a search or login is in-flight
	loading       bool
	loadingFilter models.Filter

	// stored so the results screen can be rebuilt when navigating back from detail or stats
	lastResult models.CombinedResult
	lastFilter models.Filter

	spinner spinner.Model
}

// New constructs the root App model. Credentials already in the keychain are
// used to log in without showing the form; anything else — nothing stored, a
// half-filled pair, an unreadable keychain — falls back to the login screen.
func New(cfg config.Provider, client opensearch.Searcher, store credentials.Store) App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	stored, err := store.Load()
	login := NewLoginScreen(stored.Username)
	if err != nil {
		login.errMsg = "keychain: " + err.Error()
		stored = credentials.Credentials{}
	}

	app := App{
		cfg:     cfg,
		client:  client,
		store:   store,
		stored:  stored,
		screen:  login,
		spinner: s,
	}
	if stored.Complete() {
		app.pending = stored
		app.loading = true
	}
	return app
}

// Init satisfies tea.Model. When startup credentials were found it kicks off
// the automatic login instead of waiting for the form to be submitted.
func (a App) Init() tea.Cmd {
	if a.loading {
		// The form is still initialised: it is what the user lands on when the
		// stored credentials turn out not to work.
		return tea.Batch(a.screen.Init(), a.doLogin(a.pending), a.spinner.Tick)
	}
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
		}
		if a.showHelp {
			// Any other key closes the help overlay.
			a.showHelp = false
			return a, nil
		}

	// User submitted the login form.
	case LoginSubmitMsg:
		a.loading = true
		a.pending = credentials.Credentials{Username: msg.Username, Password: msg.Password}
		return a, tea.Batch(a.doLogin(a.pending), a.spinner.Tick)

	// Login succeeded — show the results page with the filter panel ready for
	// input. A failed keychain write is reported there but does not block use.
	case LoginDoneMsg:
		a.loading = false
		a.stored = a.pending
		results := NewInitialResultsScreen(a.cfg, a.width, a.height)
		if msg.SaveErr != nil {
			results.notice = "keychain: " + msg.SaveErr.Error()
		}
		a.screen = results
		return a, tea.Batch(clearScreenCmd(), results.Init())

	// Login failed — delegate to the login screen so it can display the error.
	// An automatic login leaves the form blank, so put the password back unless
	// it is the password that was rejected.
	case loginErrMsg:
		a.loading = false
		if login, ok := a.screen.(LoginScreen); ok &&
			login.passwordInput.Value() == "" &&
			!errors.Is(msg.err, opensearch.ErrInvalidCredentials) {
			login.setPassword(a.pending.Password)
			a.screen = login
		}
		model, cmd := a.screen.Update(msg)
		a.screen = model
		return a, cmd

	// User asked the login form to forget the stored credentials.
	case ClearStoredCredentialsMsg:
		return a, a.doClearCredentials()

	case credentialsClearedMsg:
		if msg.err == nil {
			a.stored = credentials.Credentials{}
			a.pending = credentials.Credentials{}
		}
		model, cmd := a.screen.Update(msg)
		a.screen = model
		return a, cmd

	case SearchStartedMsg:
		a.loading = true
		a.loadingFilter = msg.Filter
		return a, tea.Batch(a.doSearch(msg.Filter), a.spinner.Tick)

	// Parallel search completed.
	case SearchDoneMsg:
		a.loading = false
		a.lastResult = msg.Result
		a.lastFilter = msg.Filter
		a.screen = NewResultsScreen(msg.Result, msg.Filter, a.cfg, a.width, a.height)
		return a, clearScreenCmd()

	// Spinner tick while loading.
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	// User wants to re-run the same search.
	case RefreshMsg:
		a.loading = true
		a.loadingFilter = msg.Filter
		return a, tea.Batch(a.doSearch(msg.Filter), a.spinner.Tick)

	// User selected a log entry.
	case OpenDetailMsg:
		kibanaBase := a.cfg.KibanaURL(msg.Entry.DataCenter, msg.Entry.Environment)
		a.screen = NewDetailScreen(msg.Entry, kibanaBase, a.width, a.height)
		return a, clearScreenCmd()

	// User navigates back from detail to results.
	case BackToResultsMsg:
		a.screen = NewResultsScreen(a.lastResult, a.lastFilter, a.cfg, a.width, a.height)
		return a, clearScreenCmd()

	// User wants to view statistics for the current search result — computed
	// client-side from the already-fetched entries, so no extra HTTP call.
	case ShowStatsMsg:
		a.screen = NewStatsScreen(msg.Result, msg.Filter, a.width, a.height)
		return a, clearScreenCmd()

	// User navigates back from stats to results.
	case BackFromStatsMsg:
		a.screen = NewResultsScreen(a.lastResult, a.lastFilter, a.cfg, a.width, a.height)
		return a, clearScreenCmd()
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

	summary := fmt.Sprintf("env:%s  severity:%s  app:%s",
		hi(f.Environment), hi(sev), hi(app))
	if f.Query != "" {
		summary += fmt.Sprintf("  query:%s", hi(f.Query))
	}
	if f.TraceID != "" {
		summary += fmt.Sprintf("  trace:%s", hi(f.TraceID))
	}
	summary += fmt.Sprintf("  timeframe:%s", hi(f.Timeframe))

	bar := loadingBarStyle.Width(a.width).Render(summary)
	msg := "\n  " + a.spinner.View() + " Searching…"
	return bar + msg
}

// doLogin performs the authentication request as a tea.Cmd, persisting the
// credentials once they are known to work.
func (a App) doLogin(creds credentials.Credentials) tea.Cmd {
	cfg := a.cfg
	client := a.client
	store := a.store
	stored := a.stored
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
		if err := client.Login(ctx, kibanaURL, creds.Username, creds.Password); err != nil {
			return loginErrMsg{err: err}
		}
		if creds == stored {
			return LoginDoneMsg{}
		}
		return LoginDoneMsg{SaveErr: store.Save(creds)}
	}
}

// doClearCredentials wipes the keychain entries as a tea.Cmd.
func (a App) doClearCredentials() tea.Cmd {
	store := a.store
	return func() tea.Msg {
		return credentialsClearedMsg{err: store.Clear()}
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


// clearScreenCmd returns a tea.Cmd that clears the terminal before the next render,
// preventing stale content from a previous (taller) screen from bleeding through.
func clearScreenCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
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
		helpTitleStyle.Render("crtokt — Key Bindings"),
		"",
		helpSectionStyle.Render("Filter Panel"),
		"  ctrl+s          search",
		"  tab              next field  (exits to table at last field)",
		"  shift+tab       prev field  (exits to table at first field)",
		"  enter / space  open dropdown",
		"  esc             focus results table",
		"",
		helpSectionStyle.Render("Results Table"),
		"  ↑/↓  j/k       navigate rows",
		"  enter           open detail",
		"  tab / esc        focus filter panel",
		"  shift+tab        focus filter panel (last field)",
		"  ctrl+r          refresh (re-run search)",
		"  ctrl+t          open statistics screen",
		"  /                inline text filter",
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
		"  q / ctrl+c       quit  (not while typing in a field)",
	}
	box := helpBoxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
