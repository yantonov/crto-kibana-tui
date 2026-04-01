package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap holds key bindings available on every screen.
type GlobalKeyMap struct {
	Quit key.Binding
	Help key.Binding
}

// Global is the singleton global key map.
var Global = GlobalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// FilterKeyMap holds key bindings for the filter screen.
type FilterKeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Confirm key.Binding
	Search key.Binding
}

// FilterKeys is the singleton filter key map.
var FilterKeys = FilterKeyMap{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev field"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Search: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "search"),
	),
}

// ResultsKeyMap holds key bindings for the results screen.
type ResultsKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Export  key.Binding
	Copy    key.Binding
	Refine  key.Binding
	Refresh key.Binding
	Filter  key.Binding
}

// ResultsKeys is the singleton results key map.
var ResultsKeys = ResultsKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "detail"),
	),
	Export: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "export"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy row"),
	),
	Refine: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refine search"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
}

// DetailKeyMap holds key bindings for the detail screen.
type DetailKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	Copy    key.Binding
	Open    key.Binding
	Back    key.Binding
}

// DetailKeys is the singleton detail key map.
var DetailKeys = DetailKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "toggle raw/formatted"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy JSON"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in Kibana"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "back"),
	),
}
