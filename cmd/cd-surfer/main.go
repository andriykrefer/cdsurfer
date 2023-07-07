package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// runtime.Breakpoint()
	p := tea.NewProgram(&Model{}, tea.WithOutput(os.Stderr))
	if _, err := p.Run(); err != nil {
		panic(err)
	}
	os.Exit(0)
}
