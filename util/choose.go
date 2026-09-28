package util

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("202"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	render := itemStyle.Render
	if index == m.Index() {
		render = func(values ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(values, " "))
		}
	}
	_, _ = fmt.Fprint(w, render(string(i)))
}

type model struct {
	list        list.Model
	choiceIndex int
	selected    bool
	quitting    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if _, ok := m.list.SelectedItem().(item); ok {
				m.choiceIndex = m.list.Index()
				m.selected = true
			}
			return m, tea.Quit
		}
	}

	var command tea.Cmd
	m.list, command = m.list.Update(msg)
	return m, command
}

func (m model) View() string {
	if m.selected || m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

func CreateList(title string, items []string) model {
	listItems := make([]list.Item, len(items))
	for i, value := range items {
		listItems[i] = item(value)
	}

	const defaultWidth = 20
	l := list.New(listItems, itemDelegate{}, defaultWidth, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return model{list: l}
}

// RunChooser presents choices interactively and returns the selected index.
// The interactive UI is written to stderr so stdout remains available for BibTeX.
func RunChooser(choices []string) (int, error) {
	if len(choices) == 0 {
		return -1, fmt.Errorf("chooser requires at least one option")
	}

	program := tea.NewProgram(CreateList("Choose result", choices), tea.WithOutput(os.Stderr))
	finalModel, err := program.StartReturningModel()
	if err != nil {
		return -1, fmt.Errorf("run chooser: %w", err)
	}
	return chooserSelection(finalModel)
}

func chooserSelection(finalModel tea.Model) (int, error) {
	m, ok := finalModel.(model)
	if !ok {
		return -1, fmt.Errorf("read chooser result: unexpected model %T", finalModel)
	}
	if !m.selected {
		return -1, nil
	}
	return m.choiceIndex, nil
}
