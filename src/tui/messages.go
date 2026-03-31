package tui

import "github.com/criteo/klt/src/models"

// SearchDoneMsg is sent by the app to itself when all parallel DC searches complete.
type SearchDoneMsg struct {
	Result models.CombinedResult
}
