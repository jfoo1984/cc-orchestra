package tui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	Up, Down, Enter, Filter, Rename, Pin, Archive, ShowArchived, Open, Refresh, Quit key.Binding
}

func defaultKeys() keymap {
	return keymap{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		Enter:        key.NewBinding(key.WithKeys("enter")),
		Filter:       key.NewBinding(key.WithKeys("/")),
		Rename:       key.NewBinding(key.WithKeys("n")),
		Pin:          key.NewBinding(key.WithKeys("p")),
		Archive:      key.NewBinding(key.WithKeys("a")),
		ShowArchived: key.NewBinding(key.WithKeys("A")),
		Open:         key.NewBinding(key.WithKeys("o")),
		Refresh:      key.NewBinding(key.WithKeys("r")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
	}
}
