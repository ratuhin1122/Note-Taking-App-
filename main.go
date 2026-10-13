package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type model struct {
	msg       string
	offset    int
	input     textinput.Model
	showInput bool
	notes     []string
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		// Always allow quitting with Ctrl+C or Ctrl+Q
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		}

		// When the input box is active, handle input-specific interactions
		if m.showInput {
			switch msg.String() {
			case "esc":
				m.showInput = false
				m.input.Blur()
				m.input.Reset()
				m.msg = "Cancelled new note"
				return m, nil

			case "enter":
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					m.notes = append(m.notes, val)
					m.msg = fmt.Sprintf("Note saved: %q", val)
				}
				m.showInput = false
				m.input.Blur()
				m.input.Reset()
				return m, nil
			}

			// Forward typing keys directly to the textinput model
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		// When the input box is not active
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "ctrl+n":
			m.showInput = true
			m.msg = ""
			m.input.Reset()
			cmd := m.input.Focus()
			return m, cmd
		}

	// Advance animation on each tick
	case tickMsg:
		m.offset++
		cmds = append(cmds, tickCmd())
	}

	// Allow textinput to receive cursor blink messages
	if m.showInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
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

	// Main content in the middle: shows input box when Ctrl+N is pressed
	content := ""
	if m.showInput {
		// Input box with the exact same gradient rounded border as the title card
		inputBoxStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForegroundBlend(
				lipgloss.Color("#FF79C6"),
				lipgloss.Color("#BD93F9"),
				lipgloss.Color("#8BE9FD"),
				lipgloss.Color("#BD93F9"),
				lipgloss.Color("#FF79C6"),
			).
			Padding(0, 1).
			Margin(1, 2)

		inputHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8BE9FD")).
			Render("✨ New Note (Enter to save, Esc to cancel):")

		content = inputBoxStyle.Render(inputHeader + "\n\n" + m.input.View())
	} else if len(m.notes) > 0 {
		var notesSb strings.Builder
		notesHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#BD93F9")).
			Render("📒 Your Notes:")
		notesSb.WriteString(notesHeader + "\n")

		for i, n := range m.notes {
			noteItem := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2")).
				Render(fmt.Sprintf("  %d. %s", i+1, n))
			notesSb.WriteString(noteItem + "\n")
		}

		if m.msg != "" {
			status := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B")).
				Render("\n  ✨ " + m.msg)
			notesSb.WriteString(status)
		}

		content = lipgloss.NewStyle().Margin(1, 2).Render(notesSb.String())
	} else if m.msg != "" {
		content = lipgloss.NewStyle().
			Margin(1, 2).
			Foreground(lipgloss.Color("#F1FA8C")).
			Render("📌 " + m.msg)
	}

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

	// Vertically join title card, main content view, and animated help line
	renderedUI := lipgloss.JoinVertical(
		lipgloss.Left,
		cardStyle.Render(sb.String()),
		content,
		helpStyle.Render(helpSb.String()),
	)

	v := tea.NewView(renderedUI)
	v.AltScreen = true
	return v
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Write your note here... (Enter to save, Esc to cancel)"
	ti.Prompt = " 📝 "
	ti.CharLimit = 250
	ti.SetWidth(60) // Setting width ensures the full placeholder is rendered, not just 'W'

	return model{
		msg:       "",
		offset:    0,
		input:     ti,
		showInput: false,
		notes:     []string{},
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
