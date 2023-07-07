package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// runtime.Breakpoint()
	initialPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	_ = initialPath
	model := &Model{
		// path: initialPath,
		path:   "/bin",
		width:  80,
		height: 10,
	}
	model.Ls()

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	if _, err := p.Run(); err != nil {
		panic(err)
	}
	os.Exit(0)
}
