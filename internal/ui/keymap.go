package ui

import "charm.land/bubbles/v2/key"

// keyMap declares every binding; bubbles/help can render it later.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Collapse key.Binding
	Expand   key.Binding
	Select   key.Binding
	Visual   key.Binding
	Delete   key.Binding
	Escape   key.Binding
	Filter   key.Binding
	Merged   key.Binding
	Sort     key.Binding
	Info     key.Binding
	Branches key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "move up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "move down")),
	Collapse: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("←/h", "collapse")),
	Expand:   key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("→/l", "expand")),
	Select:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	Visual:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "visual select")),
	Delete:   key.NewBinding(key.WithKeys("d", "enter"), key.WithHelp("d", "delete selected")),
	Escape:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
	Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Merged:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merged only")),
	Sort:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	Info:     key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "cycle info")),
	Branches: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branches")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
