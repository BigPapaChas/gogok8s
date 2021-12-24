package terminal

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/manifoldco/promptui"
	"github.com/pterm/pterm"
)

var ErrUserQuit = errors.New("error user quit prompt")

func Prompt(label, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   defaultValue,
		AllowEdit: true,
	}

	result, err := prompt.Run()
	if err != nil && errors.Is(err, promptui.ErrInterrupt) {
		return "", ErrUserQuit
	}
	return result, err
}

func Select(name string, choices []string) ([]string, error) {
	model := selectModel{
		title:    name,
		choices:  choices,
		selected: make(map[int]struct{}),
		userQuit: make(map[int]struct{}),
	}
	p := tea.NewProgram(model)
	if err := p.Start(); err != nil {
		return nil, err
	}
	if len(model.userQuit) > 0 {
		return nil, ErrUserQuit
	}

	var chosenOptions []string
	for i := range model.selected {
		chosenOptions = append(chosenOptions, choices[i])
	}
	return chosenOptions, nil
}

type selectModel struct {
	title    string
	choices  []string
	cursor   int
	selected map[int]struct{}
	userQuit map[int]struct{}
}

func (m selectModel) View() string {
	// The header
	s := m.title + ":\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		if checked == "x" || cursor == ">" {
			s += pterm.Green(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
		} else {
			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
		}

	}

	// The footer
	s += "\nPress c to continue, q to quit.\n"

	// Send the UI for rendering
	return s
}

func (m selectModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			m.userQuit[0] = struct{}{}
			return m, tea.Quit

		case "c":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}
