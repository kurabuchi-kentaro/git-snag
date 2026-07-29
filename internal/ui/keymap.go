package ui

import "charm.land/bubbles/v2/key"

// keyMap declares every binding; bubbles/help can render it later.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Collapse key.Binding
	Expand   key.Binding
	Filter   key.Binding
	Merged   key.Binding
	Sort     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "move up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "move down")),
	Collapse: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("←/h", "collapse")),
	Expand:   key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("→/l", "expand")),
	Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Merged:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merged only")),
	Sort:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
