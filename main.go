package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/model"
)

func main() {
	p := tea.NewProgram(model.GetInitialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
