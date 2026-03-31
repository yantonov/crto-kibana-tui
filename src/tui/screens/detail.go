package screens

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/criteo/klt/src/models"
)

// BackToResultsMsg is sent when the user presses Esc/b on the detail screen.
type BackToResultsMsg struct{}

// headerLines is the number of lines consumed by the header, separator, and
// status bar so that the viewport height can be computed correctly.
const headerLines = 10

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	detailLabelStyle = lipgloss.NewStyle().
				Width(14).
				Align(lipgloss.Right).
				Foreground(lipgloss.Color("#D1D5DB"))

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB"))

	detailSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))

	detailStatusBar = lipgloss.NewStyle().
				Background(lipgloss.Color("#1F2937")).
				Foreground(lipgloss.Color("#F9FAFB")).
				Padding(0, 1)
)

// DetailScreen shows the full content of a single log entry.
type DetailScreen struct {
	entry     models.LogEntry
	kibanaURL string

	vp      viewport.Model
	rawView bool // false = pretty-printed JSON, true = raw JSON string

	width  int
	height int
}

// NewDetailScreen constructs a DetailScreen for the given log entry.
// kibanaBaseURL is the base Kibana URL for this entry's DC/env (from config.KibanaURL);
// a trace ID query parameter is appended automatically when present.
func NewDetailScreen(entry models.LogEntry, kibanaBaseURL string, width, height int) DetailScreen {
	ds := DetailScreen{
		entry:     entry,
		kibanaURL: appendTraceID(kibanaBaseURL, entry.TraceID),
		width:     width,
		height:    height,
	}
	vpHeight := vpH(height)
	ds.vp = viewport.New(width, vpHeight)
	ds.vp.SetContent(ds.bodyContent())
	return ds
}

// Init satisfies the screen interface.
func (ds DetailScreen) Init() tea.Cmd { return nil }

// Update handles all messages for the detail screen.
func (ds DetailScreen) Update(msg tea.Msg) (DetailScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ds.width = msg.Width
		ds.height = msg.Height
		ds.vp.Width = msg.Width
		ds.vp.Height = vpH(msg.Height)
		return ds, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			return ds, func() tea.Msg { return BackToResultsMsg{} }
		case "r":
			ds.rawView = !ds.rawView
			ds.vp.SetContent(ds.bodyContent())
			return ds, nil
		case "c", "o":
			// Phase 8: clipboard copy / Kibana browser open
			return ds, nil
		}
	}

	var cmd tea.Cmd
	ds.vp, cmd = ds.vp.Update(msg)
	return ds, cmd
}

// View renders the detail screen.
func (ds DetailScreen) View() string {
	sep := detailSepStyle.Render(strings.Repeat("─", ds.width))
	return strings.Join([]string{
		ds.headerSection(),
		sep,
		ds.vp.View(),
		ds.statusBar(),
	}, "\n")
}

// ── helpers ───────────────────────────────────────────────────────────────────

// vpH computes the viewport height from the total terminal height.
func vpH(total int) int {
	h := total - headerLines
	if h < 2 {
		return 2
	}
	return h
}

func (ds DetailScreen) headerSection() string {
	e := ds.entry
	mode := "formatted"
	if ds.rawView {
		mode = "raw"
	}
	lines := []string{
		detailTitleStyle.Render(fmt.Sprintf("klt — Log Detail  [%s]", mode)),
		"",
		metaRow("Timestamp", e.Timestamp.Format("2006-01-02 15:04:05.000 UTC")),
		metaRow("Severity", e.Severity),
		metaRow("Application", e.Application),
		metaRow("DataCenter", e.DataCenter),
		metaRow("Environment", e.Environment),
		metaRow("TraceID", e.TraceID),
	}
	return strings.Join(lines, "\n")
}

func metaRow(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		detailLabelStyle.Render(label)+"  ",
		detailValueStyle.Render(value),
	)
}

func (ds DetailScreen) bodyContent() string {
	if ds.rawView {
		return ds.entry.RawJSON
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(ds.entry.RawJSON), &raw); err == nil {
		if b, err := json.MarshalIndent(raw, "", "  "); err == nil {
			return string(b)
		}
	}
	return ds.entry.RawJSON
}

func (ds DetailScreen) statusBar() string {
	toggle := "r raw"
	if ds.rawView {
		toggle = "r formatted"
	}
	keys := fmt.Sprintf("↑↓/jk scroll · %s · c copy · o kibana · esc/b back", toggle)
	return detailStatusBar.Width(ds.width).Render(keys)
}

func appendTraceID(base, traceID string) string {
	if traceID != "" {
		return base + "?q=" + traceID
	}
	return base
}
