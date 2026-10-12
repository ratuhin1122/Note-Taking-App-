package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
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
	title := "   ✨ Terminal Notes 🧠   "
	runes := []rune(title)

	// Smooth gradient for text color
	fgColors := lipgloss.Blend1D(len(runes),
		lipgloss.Color("#FF79C6"),
		lipgloss.Color("#BD93F9"),
		lipgloss.Color("#8BE9FD"),
		lipgloss.Color("#50FA7B"),
	)

	// Smooth gradient for background
	bgColors := lipgloss.Blend1D(len(runes),
		lipgloss.Color("#2C124D"),
		lipgloss.Color("#3D1966"),
		lipgloss.Color("#24144B"),
		lipgloss.Color("#181938"),
	)

	var sb strings.Builder
	for i, r := range runes {
		charStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(fgColors[i]).
			Background(bgColors[i])
		sb.WriteString(charStyle.Render(string(r)))
	}

	cardStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForegroundBlend(
			lipgloss.Color("#FF79C6"),
			lipgloss.Color("#BD93F9"),
			lipgloss.Color("#8BE9FD"),
			lipgloss.Color("#BD93F9"),
			lipgloss.Color("#FF79C6"),
		).
		Margin(1, 2)

	return tea.NewView(cardStyle.Render(sb.String()))
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
