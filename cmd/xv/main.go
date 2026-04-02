package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"xer-tui/internal/viewer"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "usage: xv <file.xer>\n")
		if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
			return
		}
		os.Exit(2)
	}

	data, err := viewer.LoadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "xv: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(viewer.NewModel(data), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "xv: %v\n", err)
		os.Exit(1)
	}
}
