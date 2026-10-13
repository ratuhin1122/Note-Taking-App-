package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type model struct {
	msg    string
	offset int
}

// tickMsg carries the timestamp for animation frames
type tickMsg time.Time

// tickCmd sends a tickMsg every 50ms to drive the animation
func tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	// Start ticking as soon as the app starts
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle key presses
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q", "q":
			return m, tea.Quit
		}

	// Advance animation on each tick
	case tickMsg:
		m.offset++
		return m, tickCmd()
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

	// Help bar text
	help := "Ctrl+N: New Note | Ctrl+L: List Notes | Ctrl+S: Save | Esc: Back | Ctrl+Q: Quit"
	helpRunes := []rune(help)

	// Flowing gradient palette for the help text
	helpPalette := lipgloss.Blend1D(len(helpRunes),
		lipgloss.Color("#50FA7B"), // Mint
		lipgloss.Color("#8BE9FD"), // Cyan
		lipgloss.Color("#BD93F9"), // Lavender
		lipgloss.Color("#FF79C6"), // Pink
		lipgloss.Color("#FFB86C"), // Orange
		lipgloss.Color("#50FA7B"), // Smooth cycle back to Mint
	)

	// Build the animated gradient text by shifting colors using m.offset
	var helpSb strings.Builder
	paletteLen := len(helpPalette)
	for i, r := range helpRunes {
		colorIdx := (i + m.offset) % paletteLen
		charStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(helpPalette[colorIdx])
		helpSb.WriteString(charStyle.Render(string(r)))
	}

	helpStyle := lipgloss.NewStyle().
		Margin(0, 2)

	// Vertically join the title card and the animated gradient help line
	renderedUI := lipgloss.JoinVertical(
		lipgloss.Left,
		cardStyle.Render(sb.String()),
		helpStyle.Render(helpSb.String()),
	)

	return tea.NewView(renderedUI)
}

func initialModel() model {
	return model{
		msg:    "H",
		offset: 0,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
