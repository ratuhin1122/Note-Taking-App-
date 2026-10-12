package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	msg string
}

func (m model) Init() tea.Cmd {
    // Just return `nil`, which means "no I/O right now, please."
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

    // Is it a key press?
    case tea.KeyPressMsg:

        // Cool, what was the actual key pressed?
        switch msg.String() {

        // These keys should exit the program.
        case "ctrl+c", "q":
            return m, tea.Quit

        
        }
    }

    return m, nil
}

func (m model) View() tea.View {
    welcome := "Welcome To Terminal Notes 🧠 "
	return tea.NewView(welcome)
}

func initialModel() model {
	return model{
		msg: "H",
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
        os.Exit(1)
    }

	
}
