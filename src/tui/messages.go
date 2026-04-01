package tui

import "github.com/criteo/klt/src/models"

// SearchDoneMsg is sent by the app to itself when all parallel DC searches complete.
type SearchDoneMsg struct {
	Result models.CombinedResult
	Filter models.Filter
}

// LoginDoneMsg is sent when the login request succeeds.
type LoginDoneMsg struct{}

// loginErrMsg is sent when the login request fails.
type loginErrMsg struct{ err error }
