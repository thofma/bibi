package util

import (
	_"encoding/json"
	"fmt"
	"io"
	_"os"
	_"reflect"
	"strings"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
  "github.com/charmbracelet/lipgloss"
	"os"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("202"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	//str := fmt.Sprintf("%d. %s", index+1, i)
	str := fmt.Sprintf("%s", i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}


type model struct {
	list     list.Model
	choice   string
	choicee  int
	quitting bool
}

func (m model) Init() tea.Cmd {
	m.choicee = -1
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
				m.choicee = m.list.Index()
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return ""
		return quitTextStyle.Render(fmt.Sprintf("%s %v? Sounds good to me.", m.choice, m.choicee))
	}
	if m.quitting {
		m.choicee = -1
		return ""
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + m.list.View()
}

func CreateList(title string, items []string) model {
	items_conv := make([]list.Item, len(items))
	for i := 0; i < len(items); i++ {
		items_conv[i] = item(items[i])
	}

	const defaultWidth = 20

	l := list.New(items_conv, itemDelegate{}, defaultWidth, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	return m
}

func RunChooser(choices []string) int {
	m := CreateList("Choose author", choices)
	p := tea.NewProgram(m)

	// StartReturningModel returns the model as a tea.Model.
	mm, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}
	// Assert the final tea.Model to our local model and print the choice.
	m, _ = mm.(model)
	if m.choice != "" {
		return m.choicee
	} else {
		return -1
	}
}
