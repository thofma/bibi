package util

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// Choice keeps a compact result label separate from its complete metadata.
type Choice struct {
	Label   string
	Details string
}

type ChoicePage struct {
	Choices   []Choice
	NextToken string
	Total     int
}

// ChooserRequest optionally loads further pages. Tokens belong to the caller;
// the chooser caches visited pages and returns an index across all loaded choices.
type ChooserRequest struct {
	Title string
	ChoicePage
	LoadPage     func(context.Context, string) (ChoicePage, error)
	Context      context.Context
	Output       io.Writer
	Confirmation bool // A single candidate opens directly in its comparison details.
}

// ChooserPageSize requests enough results to fill the terminal's results pane.
// Keep provider pages bounded, and use the chooser's default dimensions when
// output is redirected or the terminal size is unavailable.
func ChooserPageSize(output io.Writer) int {
	width, height := 80, 24
	if file, ok := output.(*os.File); ok {
		if w, h, err := term.GetSize(file.Fd()); err == nil && w > 0 && h > 0 {
			width, height = w, h
		}
	}
	return min(100, chooserResultRows(width, height))
}

func chooserResultRows(width, height int) int {
	// Header, status, navigation and paging hints each occupy one row.
	frameHeight := 1
	if width >= 24 && height >= 10 {
		frameHeight = 3
	}
	return max(1, height-4-frameHeight)
}

type detailItem struct {
	Choice
	number int
}

func (i detailItem) FilterValue() string { return i.Label }

type detailDelegate struct{ theme chooserTheme }

func (detailDelegate) Height() int                         { return 1 }
func (detailDelegate) Spacing() int                        { return 0 }
func (detailDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (delegate detailDelegate) Render(w io.Writer, m list.Model, index int, value list.Item) {
	i, ok := value.(detailItem)
	if !ok {
		return
	}
	prefix := "  "
	style := delegate.theme.base
	if index == m.Index() {
		prefix = "▸ "
		style = delegate.theme.selected
	}
	label := fmt.Sprintf("%s%d. %s", prefix, i.number, strings.Join(strings.Fields(displayText(i.Label)), " "))
	_, _ = fmt.Fprint(w, style.Width(m.Width()).Render(ansi.Truncate(label, m.Width(), "…")))
}

// Strip terminal escapes and control characters from remote metadata for display.
// The original metadata and exported BibTeX are untouched.
func displayText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, ansi.Strip(value))
}

type cachedPage struct {
	choices []Choice
	offset  int
	index   int
}

type loadedPageMsg struct {
	page ChoicePage
	err  error
}

type detailsModel struct {
	request       ChooserRequest
	pages         []cachedPage
	pageIndex     int
	nextToken     string
	total         int
	list          list.Model
	details       viewport.Model
	width, height int
	detailsFocus  bool
	compact       bool
	boxed         bool
	theme         chooserTheme
	loading       bool
	loadError     error
	choiceIndex   int
	selected      bool
	quitting      bool
}

func newDetailsModel(request ChooserRequest) detailsModel {
	output := request.Output
	if output == nil {
		output = os.Stderr
	}
	theme := newChooserTheme(output)
	l := list.New(nil, detailDelegate{theme: theme}, 80, 1)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	m := detailsModel{request: request, pages: []cachedPage{{choices: request.Choices}},
		nextToken: request.NextToken, total: request.Total, list: l, details: viewport.New(80, 1), width: 80, height: 24, theme: theme,
		detailsFocus: request.Confirmation && len(request.Choices) == 1}
	m.details.MouseWheelEnabled = false
	m.showPage(0)
	m.resize()
	return m
}

func (m detailsModel) Init() tea.Cmd { return nil }

func (m *detailsModel) showPage(index int) {
	m.loadError = nil
	m.pageIndex = index
	page := m.pages[index]
	items := make([]list.Item, len(page.choices))
	for i, choice := range page.choices {
		items[i] = detailItem{Choice: choice, number: page.offset + i + 1}
	}
	m.list.SetItems(items)
	m.list.Select(page.index)
	m.refreshDetails(true)
}

func (m *detailsModel) refreshDetails(reset bool) {
	if selected, ok := m.list.SelectedItem().(detailItem); ok {
		text := selected.Details
		if text == "" {
			text = selected.Label
		}
		m.details.SetContent(m.theme.details(text, m.details.Width))
	}
	if reset {
		m.details.GotoTop()
	}
}

func (m *detailsModel) resize() {
	m.width = max(1, m.width)
	m.height = max(1, m.height)
	resultRows := chooserResultRows(m.width, m.height)
	m.compact = m.width < 110 || m.height < 18
	m.boxed = m.width >= 24 && m.height >= 10
	frameWidth := 0
	if m.boxed {
		frameWidth = 4
	}
	if m.compact {
		m.list.SetSize(max(1, m.width-frameWidth), resultRows)
		m.details.Width, m.details.Height = m.list.Width(), m.list.Height()
	} else {
		listWidth := m.width * 45 / 100
		m.list.SetSize(listWidth-frameWidth, resultRows)
		m.details.Width, m.details.Height = m.width-listWidth-1-frameWidth, resultRows
	}
	m.refreshDetails(false)
}

func (m detailsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil
	case loadedPageMsg:
		m.loading = false
		m.loadError = msg.err
		if msg.err != nil {
			return m, nil
		}
		m.nextToken = msg.page.NextToken
		if msg.page.Total > 0 {
			m.total = msg.page.Total
		}
		if len(msg.page.Choices) == 0 {
			m.nextToken = ""
			return m, nil
		}
		last := m.pages[len(m.pages)-1]
		m.pages = append(m.pages, cachedPage{choices: msg.page.Choices, offset: last.offset + len(last.choices)})
		m.showPage(len(m.pages) - 1)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "enter":
			if _, ok := m.list.SelectedItem().(detailItem); ok {
				m.choiceIndex = m.pages[m.pageIndex].offset + m.list.Index()
				m.selected = true
			}
			return m, tea.Quit
		case "tab":
			m.detailsFocus = !m.detailsFocus
			return m, nil
		case "p":
			if m.pageIndex > 0 {
				m.pages[m.pageIndex].index = m.list.Index()
				m.showPage(m.pageIndex - 1)
			}
			return m, nil
		case "n":
			m.pages[m.pageIndex].index = m.list.Index()
			if m.pageIndex+1 < len(m.pages) {
				m.showPage(m.pageIndex + 1)
				return m, nil
			}
			if m.nextToken != "" && m.request.LoadPage != nil {
				m.loading, m.loadError = true, nil
				ctx, token, load := m.request.Context, m.nextToken, m.request.LoadPage
				if ctx == nil {
					ctx = context.Background()
				}
				return m, func() tea.Msg {
					page, err := load(ctx, token)
					return loadedPageMsg{page: page, err: err}
				}
			}
			return m, nil
		}
	}
	if m.loading {
		return m, nil
	}
	var command tea.Cmd
	if m.detailsFocus {
		m.details, command = m.details.Update(msg)
	} else {
		index := m.list.Index()
		m.list, command = m.list.Update(msg)
		if m.list.Index() != index {
			m.refreshDetails(true)
		}
	}
	return m, command
}

func (m detailsModel) View() string {
	if m.selected || m.quitting {
		return ""
	}
	line := func(value string) string {
		return ansi.Truncate(strings.ReplaceAll(value, "\n", " "), m.width, "…")
	}
	page := m.pages[m.pageIndex]
	status := fmt.Sprintf("Results %d–%d · page %d", page.offset+1, page.offset+len(page.choices), m.pageIndex+1)
	if m.total > 0 {
		status += fmt.Sprintf(" · %d matches", m.total)
	}
	focus := "DETAILS · Tab to scroll"
	if m.detailsFocus {
		focus = fmt.Sprintf("DETAILS · %.0f%% · Tab for results", m.details.ScrollPercent()*100)
	}
	resultsTitle := fmt.Sprintf("RESULTS · %d/%d", m.list.Index()+1, len(page.choices))
	results, details := line(resultsTitle)+"\n"+m.list.View(), line(focus)+"\n"+m.details.View()
	if m.boxed {
		results = m.theme.box(resultsTitle, m.list.View(), m.list.Width()+4, !m.detailsFocus)
		details = m.theme.box(focus, m.details.View(), m.details.Width+4, m.detailsFocus)
	}
	var body string
	if m.compact {
		body = results
		if m.detailsFocus {
			body = details
		}
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, results, " ", details)
	}
	action := "select"
	if m.request.Confirmation {
		action = "confirm"
	}
	navigation := m.theme.shortcut("↑/↓", "move") + "  " + m.theme.shortcut("Tab", "details") + "  " +
		m.theme.shortcut("Enter", action) + "  " + m.theme.shortcut("q", "cancel")
	if m.detailsFocus {
		navigation = m.theme.shortcut("↑/↓ PgUp/PgDn", "scroll") + "  " + m.theme.shortcut("Tab", "results") + "  " +
			m.theme.shortcut("Enter", action) + "  " + m.theme.shortcut("q", "cancel")
	}
	if m.width < 70 {
		navigation = m.theme.shortcut("↑/↓", "") + m.theme.shortcut("Tab", "details") + "  " +
			m.theme.shortcut("↵", action) + "  " + m.theme.shortcut("q", "quit")
		if m.detailsFocus {
			navigation = m.theme.shortcut("↑/↓", "scroll") + " " + m.theme.shortcut("Tab", "results") + " " + m.theme.shortcut("↵", "") + m.theme.shortcut("q", "")
		}
	}
	pages := ""
	if m.pageIndex > 0 {
		pages = m.theme.shortcut("p", "previous page")
	}
	if m.pageIndex+1 < len(m.pages) || (m.nextToken != "" && m.request.LoadPage != nil) {
		if pages != "" {
			pages += " · "
		}
		pages += m.theme.shortcut("n", "next page")
	}
	if m.loading {
		pages = m.theme.accent.Render("◌ Loading next page…") + "  " + m.theme.shortcut("q", "cancel")
	} else if m.loadError != nil {
		pages = m.theme.shortcut("n", "retry") + " · " + m.theme.warm.Render("Page failed: "+displayText(m.loadError.Error()))
	}
	header := m.theme.brand.Render(" bibi ") + "  " + m.theme.accent.Render(displayText(m.request.Title))
	view := strings.Join([]string{line(header), line(m.theme.muted.Render(status)), body, line(navigation), line(pages)}, "\n")
	return lipgloss.NewStyle().MaxHeight(m.height).Render(view)
}

// RunDetailedChooser writes its interactive UI to stderr and never fetches BibTeX.
func RunDetailedChooser(request ChooserRequest) (int, error) {
	return runDetailedChooser(request, openChooserTerminal)
}

func runDetailedChooser(request ChooserRequest, openInput func() (io.ReadCloser, error)) (int, error) {
	if len(request.Choices) == 0 {
		return -1, fmt.Errorf("chooser requires at least one option")
	}
	ctx := request.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	request.Context = ctx
	if err := ctx.Err(); err != nil {
		return -1, err
	}
	input, err := openInput()
	if err != nil {
		return -1, fmt.Errorf("%w: rerun in an interactive terminal to review and confirm the entry: %w", ErrInteractiveTerminalUnavailable, err)
	}
	defer input.Close()
	output := request.Output
	if output == nil {
		output = os.Stderr
	}
	program := tea.NewProgram(newDetailsModel(request), tea.WithInput(input), tea.WithOutput(output), tea.WithContext(ctx), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		return -1, fmt.Errorf("run chooser: %w", err)
	}
	m, ok := final.(detailsModel)
	if !ok {
		return -1, fmt.Errorf("read chooser result: unexpected model %T", final)
	}
	if !m.selected {
		return -1, nil
	}
	return m.choiceIndex, nil
}
