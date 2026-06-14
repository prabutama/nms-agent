package tui

import "github.com/charmbracelet/bubbles/key"

type tuiKeyMap struct {
	Quit           key.Binding
	Help           key.Binding
	SelectUp       key.Binding
	SelectDown     key.Binding
	FocusNext      key.Binding
	FocusPrev      key.Binding
	ToggleFullHelp key.Binding
}

func newTUIKeyMap() tuiKeyMap {
	return tuiKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "help"),
		),
		SelectUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "select device"),
		),
		SelectDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "select device"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "focus"),
		),
		ToggleFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more keys"),
		),
	}
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.SelectUp, k.SelectDown, k.Quit}
}

func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.SelectUp, k.SelectDown, k.FocusNext, k.FocusPrev, k.Quit, k.Help}}
}
