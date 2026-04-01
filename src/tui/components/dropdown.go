package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// innerWidth is the text content width inside the dropdown border box.
const innerWidth = 50

var (
	styleFocused = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			PaddingLeft(1).PaddingRight(1)

	styleBlurred = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6B7280")).
			PaddingLeft(1).PaddingRight(1)

	styleCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
)

// Option is one selectable entry in a Dropdown.
type Option struct {
	Label string
	Value string
}

// Dropdown is a collapsible selector component.
type Dropdown struct {
	options  []Option
	cursor   int
	expanded bool
	focused  bool
}

// New creates a Dropdown pre-populated with options.
func New(options []Option) Dropdown {
	return Dropdown{options: options}
}

// SetFocused controls whether this is the active field.
func (d *Dropdown) SetFocused(v bool) { d.focused = v }

// Collapse closes the option list without changing the selection.
func (d *Dropdown) Collapse() { d.expanded = false }

// IsExpanded reports whether the option list is visible.
func (d Dropdown) IsExpanded() bool { return d.expanded }

// Selected returns the currently chosen option.
func (d Dropdown) Selected() Option {
	if len(d.options) == 0 {
		return Option{}
	}
	return d.options[d.cursor]
}

// SetByValue pre-selects the option whose Value equals v.
func (d *Dropdown) SetByValue(v string) {
	for i, o := range d.options {
		if o.Value == v {
			d.cursor = i
			return
		}
	}
}

// Update processes keyboard input.
// The second return value is true when the key was consumed (parent should
// skip its own handling for that key).
func (d Dropdown) Update(msg tea.Msg) (Dropdown, bool) {
	if !d.focused {
		return d, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, false
	}

	// Tab and Shift+Tab collapse the list and propagate to the parent.
	if key.String() == "tab" || key.String() == "shift+tab" {
		d.expanded = false
		return d, false
	}

	if !d.expanded {
		if key.String() == "enter" || key.String() == " " {
			d.expanded = true
			return d, true
		}
		return d, false
	}

	// Expanded — handle navigation; consume all keys.
	switch key.String() {
	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}
	case "down", "j":
		if d.cursor < len(d.options)-1 {
			d.cursor++
		}
	case "enter", " ":
		d.expanded = false
	case "esc":
		d.expanded = false
	}
	return d, true
}

// View renders the dropdown (collapsed or expanded).
func (d Dropdown) View() string {
	label := "(none)"
	if len(d.options) > 0 {
		label = d.options[d.cursor].Label
	}

	header := padOrTrunc(label, innerWidth) + " ▾"

	var sb strings.Builder
	sb.WriteString(header)

	if d.expanded {
		for i, o := range d.options {
			sb.WriteByte('\n')
			if i == d.cursor {
				sb.WriteString("▸ " + styleCursor.Render(o.Label))
			} else {
				sb.WriteString("  " + o.Label)
			}
		}
	}

	if d.focused {
		return styleFocused.Width(innerWidth + 2).Render(sb.String())
	}
	return styleBlurred.Width(innerWidth + 2).Render(sb.String())
}

// padOrTrunc pads s with spaces or truncates it to exactly n runes.
func padOrTrunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) >= n {
		return string(runes[:n])
	}
	return s + strings.Repeat(" ", n-len(runes))
}
