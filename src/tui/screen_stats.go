package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yantonov/crtokt/src/models"
)

var (
	statsTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	statsSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))

	statsPanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#D1D5DB"))
)

// statsFixedLines is the number of lines consumed by the title, summary,
// two separators, bottom-panel title row, and status bar — everything except
// the histogram viewport and the bottom-panel data rows.
const statsFixedLines = 6

// timeBucket holds the entry count for a single time interval.
type timeBucket struct {
	epochMs  int64
	docCount int
}

// termBucket holds the entry count for a single label (severity or DC).
type termBucket struct {
	key      string
	docCount int
}

// StatsScreen shows a histogram and breakdowns derived from the already-fetched
// CombinedResult — no extra OpenSearch call is made.
type StatsScreen struct {
	result  models.CombinedResult
	filter  models.Filter
	buckets []timeBucket
	bySev   []termBucket
	byDC    []termBucket

	vp     viewport.Model
	width  int
	height int
}

// NewStatsScreen constructs a StatsScreen from an existing search result.
// All statistics are computed client-side from result.Entries.
func NewStatsScreen(result models.CombinedResult, filter models.Filter, width, height int) StatsScreen {
	ss := StatsScreen{
		result:  result,
		filter:  filter,
		buckets: buildTimeBuckets(result.Entries, filter.Timeframe),
		bySev:   buildTermBuckets(result.Entries, func(e models.LogEntry) string { return e.Severity }),
		byDC:    buildTermBuckets(result.Entries, func(e models.LogEntry) string { return e.DataCenter }),
		width:   width,
		height:  height,
	}
	ss.vp = viewport.New(width, ss.vpHeight())
	ss.vp.SetContent(ss.histogramContent())
	return ss
}

// Init satisfies the Screen interface.
func (ss StatsScreen) Init() tea.Cmd { return nil }

// Update handles all messages for the stats screen.
func (ss StatsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ss.width = msg.Width
		ss.height = msg.Height
		ss.vp.Width = msg.Width
		ss.vp.Height = ss.vpHeight()
		ss.vp.SetContent(ss.histogramContent())
		return ss, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			return ss, func() tea.Msg { return BackFromStatsMsg{} }
		}
	}

	var cmd tea.Cmd
	ss.vp, cmd = ss.vp.Update(msg)
	return ss, cmd
}

// View renders the stats screen.
func (ss StatsScreen) View() string {
	sep := statsSepStyle.Render(strings.Repeat("─", ss.width))
	return strings.Join([]string{
		ss.titleLine(),
		ss.summaryLine(),
		sep,
		ss.vp.View(),
		sep,
		ss.bottomSection(),
		ss.statusBar(),
	}, "\n")
}

// ── layout helpers ────────────────────────────────────────────────────────────

func (ss StatsScreen) bottomRows() int {
	n := len(ss.bySev)
	if m := len(ss.byDC); m > n {
		n = m
	}
	if n > 7 {
		n = 7
	}
	return n
}

func (ss StatsScreen) vpHeight() int {
	h := ss.height - statsFixedLines - ss.bottomRows()
	if h < 3 {
		return 3
	}
	return h
}

// ── rendering ─────────────────────────────────────────────────────────────────

func (ss StatsScreen) titleLine() string {
	return statsTitleStyle.Render("crtokt — Statistics")
}

func (ss StatsScreen) summaryLine() string {
	f := ss.filter
	hi := filterSummaryHighlight.Render

	app := "all"
	if f.Application != "" {
		app = f.Application
	}
	sev := "all"
	if f.Severity >= 0 {
		sev = models.SeverityLabel(f.Severity)
	}

	total := ss.result.TotalHits
	fetched := len(ss.result.Entries)

	totalStr := strconv.Itoa(total)
	suffix := ""
	if fetched < total && fetched > 0 {
		suffix = fmt.Sprintf(" (histogram from top %d)", fetched)
	}

	line := fmt.Sprintf("env:%s  severity:%s  app:%s  timeframe:%s   %s total%s",
		hi(f.Environment), hi(sev), hi(app), hi(f.Timeframe), hi(totalStr), suffix)

	if len(ss.result.DCErrors) > 0 {
		line += fmt.Sprintf("  (%d DC error(s))", len(ss.result.DCErrors))
	}

	return filterSummaryStyle.Width(ss.width).Render(line)
}

func (ss StatsScreen) histogramContent() string {
	if len(ss.buckets) == 0 {
		return "  no data"
	}

	maxCount := 0
	for _, b := range ss.buckets {
		if b.docCount > maxCount {
			maxCount = b.docCount
		}
	}

	labelW := bucketLabelWidth(ss.filter.Timeframe)
	countW := len(strconv.Itoa(maxCount))
	// 2 indent + labelW + 2 gap + barAreaW + 2 gap + countW = width
	barAreaW := ss.width - 2 - labelW - 2 - 2 - countW
	if barAreaW < 1 {
		barAreaW = 1
	}

	lines := make([]string, 0, len(ss.buckets))
	for _, b := range ss.buckets {
		label := formatBucketLabel(b.epochMs, ss.filter.Timeframe)
		barLen := 0
		if maxCount > 0 {
			barLen = (b.docCount * barAreaW) / maxCount
		}
		bar := strings.Repeat("█", barLen)
		line := fmt.Sprintf("  %-*s  %-*s  %*d", labelW, label, barAreaW, bar, countW, b.docCount)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (ss StatsScreen) bottomSection() string {
	rows := ss.bottomRows()
	halfW := ss.width / 2
	left := renderTermPanel("By severity", ss.bySev, halfW, rows)
	right := renderTermPanel("By datacenter", ss.byDC, ss.width-halfW, rows)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (ss StatsScreen) statusBar() string {
	return StatusBar.Width(ss.width).Render("↑↓/jk scroll · esc/b back · ctrl+c quit")
}

// ── computation helpers ───────────────────────────────────────────────────────

func buildTimeBuckets(entries []models.LogEntry, timeframe string) []timeBucket {
	step := intervalDuration(timeframe)
	dur := timeframeDuration(timeframe)
	stepMs := step.Milliseconds()

	now := time.Now().UTC()
	endMs := (now.UnixMilli() / stepMs) * stepMs
	startMs := endMs - dur.Milliseconds()

	numBuckets := int(dur.Milliseconds() / stepMs)
	if numBuckets < 1 {
		numBuckets = 1
	}

	counts := make([]int, numBuckets)
	for _, e := range entries {
		ts := e.Timestamp.UnixMilli()
		if ts < startMs {
			continue
		}
		idx := int((ts - startMs) / stepMs)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		if idx >= 0 {
			counts[idx]++
		}
	}

	buckets := make([]timeBucket, numBuckets)
	for i := range buckets {
		buckets[i] = timeBucket{
			epochMs:  startMs + int64(i)*stepMs,
			docCount: counts[i],
		}
	}
	return buckets
}

func buildTermBuckets(entries []models.LogEntry, key func(models.LogEntry) string) []termBucket {
	m := make(map[string]int)
	for _, e := range entries {
		if k := key(e); k != "" {
			m[k]++
		}
	}
	result := make([]termBucket, 0, len(m))
	for k, v := range m {
		result = append(result, termBucket{key: k, docCount: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].docCount > result[j].docCount
	})
	return result
}

// intervalDuration returns the per-bucket step size chosen to produce
// roughly 30–40 buckets across the full timeframe window.
//
//	15m  / 30s  = 30 buckets
//	30m  / 1m   = 30 buckets
//	1h   / 2m   = 30 buckets
//	3h   / 5m   = 36 buckets
//	6h   / 10m  = 36 buckets
//	24h  / 30m  = 48 buckets
//	2d   / 1h   = 48 buckets
//	7d   / 4h   = 42 buckets
func intervalDuration(tf string) time.Duration {
	switch tf {
	case "15m":
		return 30 * time.Second
	case "30m":
		return 1 * time.Minute
	case "1h":
		return 2 * time.Minute
	case "3h":
		return 5 * time.Minute
	case "6h":
		return 10 * time.Minute
	case "24h":
		return 30 * time.Minute
	case "2d":
		return 1 * time.Hour
	case "7d":
		return 4 * time.Hour
	default:
		return 5 * time.Minute
	}
}

func timeframeDuration(tf string) time.Duration {
	switch tf {
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "3h":
		return 3 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "2d":
		return 48 * time.Hour
	case "7d":
		return 168 * time.Hour
	default:
		return 1 * time.Hour
	}
}

func bucketLabelWidth(tf string) int {
	switch tf {
	case "15m":
		return 8 // "15:04:05"
	case "2d", "7d":
		return 11 // "01-02 15:04"
	default:
		return 5 // "15:04"
	}
}

func formatBucketLabel(epochMs int64, timeframe string) string {
	t := time.UnixMilli(epochMs).UTC()
	switch timeframe {
	case "15m":
		return t.Format("15:04:05")
	case "2d", "7d":
		return t.Format("01-02 15:04")
	default:
		return t.Format("15:04")
	}
}

func renderTermPanel(title string, buckets []termBucket, width, rows int) string {
	maxCount := 0
	maxKeyLen := 0
	for _, b := range buckets {
		if b.docCount > maxCount {
			maxCount = b.docCount
		}
		if len(b.key) > maxKeyLen {
			maxKeyLen = len(b.key)
		}
	}
	if maxKeyLen > width/3 {
		maxKeyLen = width / 3
	}
	if maxKeyLen < 1 {
		maxKeyLen = 1
	}

	countW := len(strconv.Itoa(maxCount))
	if countW < 1 {
		countW = 1
	}
	// 2 indent + maxKeyLen + 2 gap + barAreaW + 2 gap + countW = width
	barAreaW := width - 2 - maxKeyLen - 2 - 2 - countW
	if barAreaW < 1 {
		barAreaW = 1
	}

	n := rows
	if n > len(buckets) {
		n = len(buckets)
	}

	lines := make([]string, 0, n+1)
	lines = append(lines, statsPanelTitleStyle.Render("  "+title))

	for _, b := range buckets[:n] {
		barLen := 0
		if maxCount > 0 {
			barLen = (b.docCount * barAreaW) / maxCount
		}
		bar := strings.Repeat("█", barLen)
		key := b.key
		if len([]rune(key)) > maxKeyLen {
			key = string([]rune(key)[:maxKeyLen])
		}
		line := fmt.Sprintf("  %-*s  %-*s  %*d", maxKeyLen, key, barAreaW, bar, countW, b.docCount)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
