package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Global directory where all .md notes will be saved across all OS platforms
var notesDir string

func init() {
	// Resolves user home directory on Windows (C:\Users\<user>), macOS (/Users/<user>), Linux (/home/<user>)
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	notesDir = filepath.Join(home, ".terminal-notes")

	// Use os.Stat() to check if the directory already exists
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create notes directory: %v\n", err)
		}
	}
}

// openFolderInExplorer opens the notes folder in the native file manager (Explorer, Finder, xdg-open)
func openFolderInExplorer(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// loadSavedNotes reads all markdown (.md) notes from the user's notes directory
func loadSavedNotes() []string {
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		return nil
	}
	var list []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			list = append(list, entry.Name())
		}
	}
	return list
}

// noteItem implements bubbles list.DefaultItem interface
type noteItem struct {
	filename string
}

func (i noteItem) Title() string       { return i.filename + " ↗" }
func (i noteItem) Description() string { return "" }
func (i noteItem) FilterValue() string { return i.filename }

type sessionState int

const (
	stateDefault sessionState = iota
	stateFilenameInput
	stateContentEditing
)

// clearStatusMsg is fired by a timer to auto-dismiss status messages after 5 seconds
type clearStatusMsg struct {
	id int
}

type model struct {
	state           sessionState
	msg             string
	statusID        int
	offset          int
	filenameInput   textinput.Model
	contentArea     textarea.Model
	notesList       list.Model
	currentFilename string
	savedNotes      []string
}

// setStatus displays a message and automatically clears it after 5 seconds
func (m *model) setStatus(text string) tea.Cmd {
	m.msg = text
	m.statusID++
	id := m.statusID
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{id: id}
	})
}

// openNote loads note content from disk into the textarea editor
func (m *model) openNote(filename string) tea.Cmd {
	targetPath := filepath.Join(notesDir, filename)
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return m.setStatus(fmt.Sprintf("Error opening note: %v", err))
	}

	m.currentFilename = filename
	m.contentArea.Reset()
	m.contentArea.SetValue(string(content))
	m.state = stateContentEditing
	m.msg = ""
	return m.contentArea.Focus()
}

// tickMsg carries timestamp to drive the color wave animation
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Auto-dismiss status notification when 5-second timer expires
	case clearStatusMsg:
		if msg.id == m.statusID {
			m.msg = ""
		}
		return m, nil

	// Mouse click support
	case tea.MouseClickMsg:
		if m.state == stateDefault {
			// Clicking on row 6 (the folder path header) opens the folder in Explorer/Finder
			if msg.Y == 6 {
				if err := openFolderInExplorer(notesDir); err != nil {
					return m, m.setStatus(fmt.Sprintf("Error opening folder: %v", err))
				}
				return m, m.setStatus("Opened notes folder in file explorer")
			}

			// Mouse events are disabled for the notes list; keyboard navigation (Enter, arrows) is used instead
			return m, nil
		}

	case tea.KeyPressMsg:
		// Global exit shortcuts
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		}

		// Handle key actions depending on current state
		switch m.state {

		case stateDefault:
			// If user is currently typing inside the list's fuzzy search filter
			if m.notesList.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.notesList, cmd = m.notesList.Update(msg)
				return m, cmd
			}

			switch msg.String() {
			case "q":
				return m, tea.Quit

			case "o", "ctrl+o":
				// Open notes folder in file explorer
				if err := openFolderInExplorer(notesDir); err != nil {
					return m, m.setStatus(fmt.Sprintf("Error opening folder: %v", err))
				}
				return m, m.setStatus("Opened notes folder in file explorer")

			case "enter":
				// Open the currently selected note in the bubbles list
				if selected, ok := m.notesList.SelectedItem().(noteItem); ok {
					cmd := m.openNote(selected.filename)
					return m, cmd
				}
				return m, nil

			case "ctrl+n":
				m.state = stateFilenameInput
				m.msg = ""
				m.filenameInput.Reset()
				return m, m.filenameInput.Focus()
			}

			// Forward navigation keys (Up, Down, PageUp, PageDown, /) to bubbles list
			var cmd tea.Cmd
			m.notesList, cmd = m.notesList.Update(msg)
			return m, cmd

		case stateFilenameInput:
			switch msg.String() {
			case "esc":
				m.state = stateDefault
				m.filenameInput.Blur()
				m.filenameInput.Reset()
				return m, m.setStatus("Cancelled new note")

			case "enter":
				rawName := strings.TrimSpace(m.filenameInput.Value())
				if rawName == "" {
					rawName = fmt.Sprintf("note_%s", time.Now().Format("20060102_150405"))
				}

				// Guarantee .md extension
				if !strings.HasSuffix(strings.ToLower(rawName), ".md") {
					rawName += ".md"
				}

				// Check for DUPLICATE filename using os.Stat()
				targetPath := filepath.Join(notesDir, rawName)
				if _, err := os.Stat(targetPath); err == nil {
					return m, m.setStatus(fmt.Sprintf("⚠️ File %q already exists! Choose another name.", rawName))
				}

				m.currentFilename = rawName
				m.state = stateContentEditing
				m.msg = ""
				m.filenameInput.Blur()
				m.contentArea.Reset()
				return m, m.contentArea.Focus()
			}

			var cmd tea.Cmd
			m.filenameInput, cmd = m.filenameInput.Update(msg)
			return m, cmd

		case stateContentEditing:
			switch msg.String() {
			case "esc":
				m.state = stateDefault
				m.contentArea.Blur()
				m.contentArea.Reset()
				return m, m.setStatus("Closed note editor")

			case "ctrl+s":
				// Save note file to notes directory
				targetPath := filepath.Join(notesDir, m.currentFilename)
				content := m.contentArea.Value()

				var cmd tea.Cmd
				err := os.WriteFile(targetPath, []byte(content), 0644)
				if err != nil {
					cmd = m.setStatus(fmt.Sprintf("Error saving note: %v", err))
				} else {
					// Refresh notes and update bubbles list
					m.savedNotes = loadSavedNotes()
					newItems := make([]list.Item, len(m.savedNotes))
					for i, n := range m.savedNotes {
						newItems[i] = noteItem{filename: n}
					}
					m.notesList.SetItems(newItems)

					cmd = m.setStatus(fmt.Sprintf("Saved note to %s", m.currentFilename))
				}

				m.state = stateDefault
				m.contentArea.Blur()
				m.contentArea.Reset()
				return m, cmd
			}

			var cmd tea.Cmd
			m.contentArea, cmd = m.contentArea.Update(msg)
			return m, cmd
		}

	case tickMsg:
		m.offset++
		cmds = append(cmds, tickCmd())
	}

	// Update components for blinking cursors
	if m.state == stateFilenameInput {
		var cmd tea.Cmd
		m.filenameInput, cmd = m.filenameInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else if m.state == stateContentEditing {
		var cmd tea.Cmd
		m.contentArea, cmd = m.contentArea.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else if m.state == stateDefault {
		// Forward component messages (such as list.FilterMatchesMsg and cursor blinks) to the list
		if _, isTick := msg.(tickMsg); !isTick {
			var cmd tea.Cmd
			m.notesList, cmd = m.notesList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
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

	// Reusable gradient box style matching title box
	gradientBoxStyle := lipgloss.NewStyle().
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

	// Dynamic middle content based on application state
	content := ""

	switch m.state {

	case stateFilenameInput:
		header := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8BE9FD")).
			Render("📁 Step 1: Note File Name (.md)")

		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("Press Enter to continue • Esc to cancel")

		errNotice := ""
		if m.msg != "" {
			errNotice = "\n\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5555")).
				Bold(true).
				Render(m.msg)
		}

		content = gradientBoxStyle.Render(header + "\n\n" + m.filenameInput.View() + errNotice + "\n\n" + hint)

	case stateContentEditing:
		header := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#50FA7B")).
			Render(fmt.Sprintf("📝 Editing: %s", m.currentFilename))

		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("Press Ctrl+S to save • Esc to close")

		content = gradientBoxStyle.Render(header + "\n\n" + m.contentArea.View() + "\n\n" + hint)

	case stateDefault:
		var notesSb strings.Builder

		if len(m.savedNotes) > 0 {
			// Clickable folder path header
			folderBadge := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#8BE9FD")).
				Render(notesDir + " ↗")

			notesHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#BD93F9")).
				Render(fmt.Sprintf("📒 Saved Notes (%d in ", len(m.savedNotes))) +
				folderBadge +
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BD93F9")).Render("):")

			if m.notesList.IsFiltered() {
				filterBadge := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#50FA7B")).
					Render(fmt.Sprintf(" [Filter: %q]", m.notesList.FilterValue()))
				notesHeader += filterBadge
			}

			notesSb.WriteString(notesHeader + "\n\n")

			// Render the bubbles list component
			notesSb.WriteString(m.notesList.View())
		} else {
			emptyNotice := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4")).
				Render("  No notes saved yet. Press Ctrl+N to create your first note!")
			notesSb.WriteString(emptyNotice + "\n")
		}

		if m.msg != "" {
			status := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B")).
				Render("\n✨ " + m.msg)
			notesSb.WriteString(status)
		}

		content = lipgloss.NewStyle().Margin(1, 2).Render(notesSb.String())
	}

	// Dynamic help text
	help := "Ctrl+N: New Note | Enter: Open Note | /: Filter | O: Open Folder | Ctrl+Q: Quit"
	if m.state == stateFilenameInput {
		help = "Enter: Continue to Editor | Esc: Cancel | Ctrl+Q: Quit"
	} else if m.state == stateContentEditing {
		help = "Ctrl+S: Save Note | Esc: Close Editor | Ctrl+Q: Quit"
	} else if m.notesList.FilterState() == list.Filtering {
		help = "Enter: Apply Filter | Esc: Cancel Filter | Ctrl+Q: Quit"
	} else if m.notesList.FilterState() == list.FilterApplied {
		help = "Enter: Open Note | Esc: Clear Filter | /: Edit Filter | Ctrl+Q: Quit"
	}
	helpRunes := []rune(help)

	// Animated gradient palette for the help text
	helpPalette := lipgloss.Blend1D(len(helpRunes),
		lipgloss.Color("#50FA7B"), // Mint
		lipgloss.Color("#8BE9FD"), // Cyan
		lipgloss.Color("#BD93F9"), // Lavender
		lipgloss.Color("#FF79C6"), // Pink
		lipgloss.Color("#FFB86C"), // Orange
		lipgloss.Color("#50FA7B"), // Seamless loop
	)

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

	// Vertically join title card, main content, and animated help line
	renderedUI := lipgloss.JoinVertical(
		lipgloss.Left,
		cardStyle.Render(sb.String()),
		content,
		helpStyle.Render(helpSb.String()),
	)

	v := tea.NewView(renderedUI)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func initialModel() model {
	// Filename input setup
	ti := textinput.New()
	ti.Placeholder = "e.g. daily-standup, quick-ideas"
	ti.Prompt = " 📂 Filename: "
	ti.CharLimit = 100
	ti.SetWidth(50)

	// Multi-line note text area setup
	ta := textarea.New()
	ta.Placeholder = "Write your markdown note here..."
	ta.Prompt = ""             // Remove thick bar on the left
	ta.ShowLineNumbers = false // Remove line numbers

	// Remove solid black background and keep styling clean and transparent
	styles := ta.Styles()
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Blurred.CursorLine = lipgloss.NewStyle()
	styles.Focused.Base = lipgloss.NewStyle()
	styles.Blurred.Base = lipgloss.NewStyle()
	ta.SetStyles(styles)

	ta.SetWidth(65)
	ta.SetHeight(10)

	// Load initial notes and initialize Bubble Tea list component
	saved := loadSavedNotes()
	items := make([]list.Item, len(saved))
	for i, n := range saved {
		items[i] = noteItem{filename: n}
	}

	// Create Bubble Tea simple list delegate
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)

	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#50FA7B")).
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true).
		Padding(0, 0, 0, 1)

	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		Padding(0, 0, 0, 2)

	notesList := list.New(items, delegate, 65, 8)
	notesList.SetShowTitle(false)
	notesList.SetShowStatusBar(false)
	notesList.SetShowHelp(false)
	notesList.SetFilteringEnabled(true)
	notesList.InfiniteScrolling = true

	return model{
		state:         stateDefault,
		msg:           "",
		offset:        0,
		filenameInput: ti,
		contentArea:   ta,
		notesList:     notesList,
		savedNotes:    saved,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
