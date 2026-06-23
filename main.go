package main

import (
	"fmt"
	"os"
	"stack-stitcher/src/model"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(model.GetInitialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
