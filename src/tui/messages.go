package tui

import "github.com/criteo/klt/src/models"

// ── messages sent from screens up to App ─────────────────────────────────────

// LoginSubmitMsg is sent when the user submits the login form.
type LoginSubmitMsg struct {
	Username string
	Password string
}

// SearchStartedMsg is sent when the user triggers a search from the filter screen.
type SearchStartedMsg struct {
	Filter models.Filter
}

// RefreshMsg is sent when the user wants to re-run the same search.
type RefreshMsg struct {
	Filter models.Filter
}

// OpenDetailMsg is sent when the user selects a log entry.
type OpenDetailMsg struct {
	Entry models.LogEntry
}

// BackToResultsMsg is sent when the user presses Esc/b on the detail screen.
type BackToResultsMsg struct{}

// ── messages sent from App to itself ─────────────────────────────────────────

// SearchDoneMsg is sent by the app to itself when all parallel DC searches complete.
type SearchDoneMsg struct {
	Result models.CombinedResult
	Filter models.Filter
}

// LoginDoneMsg is sent when the login request succeeds.
type LoginDoneMsg struct{}

// loginErrMsg is sent when the login request fails.
type loginErrMsg struct{ err error }
