package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	_"net/url"
	"log"
	"flag"
	"os"
	_"log"
	_"reflect"
	"strings"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
  "github.com/charmbracelet/lipgloss"


)

/*
    Stuff for the menu
														*/

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

//type model struct {
//	cursor int
//	choice int
//	choices []string
//}
//
//func (m model) Init() tea.Cmd {
//	return nil
//}
//
//func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	switch msg := msg.(type) {
//	case tea.KeyMsg:
//		switch msg.String() {
//		case "ctrl+c", "q", "esc":
//			return m, tea.Quit
//
//		case "enter":
//			// Send the choice on the channel and exit.
//			m.choice = m.cursor
//			return m, tea.Quit
//
//		case "down", "j":
//			m.cursor++
//			if m.cursor >= len(m.choices) {
//				m.cursor = 0
//			}
//
//		case "up", "k":
//			m.cursor--
//			if m.cursor < 0 {
//				m.cursor = len(m.choices) - 1
//			}
//		}
//
//	}
//
//	return m, nil
//}
//
//func (m model) View() string {
//	s := strings.Builder{}
//	s.WriteString("Multiple entries found:\n")
//
//	for i := 0; i < len(m.choices); i++ {
//		if m.cursor == i {
//			s.WriteString("-> ")
//		} else {
//			s.WriteString("   ")
//		}
//		s.WriteString(m.choices[i])
//		s.WriteString("\n")
//	}
//	s.WriteString("\n(press q to quit)\n")
//
//	return s.String()
//}

func detect_doi(name string) (bool, string) {
	return true, ""
}

func doi_to_bibtex(doi string) string {
	url := "http://dx.doi.org/10.1017/fms.2022.74"

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	// Set the Accept header
	req.Header.Set("Accept", "application/citeproc+json")

	// Create an HTTP client that follows redirects
	client := &http.Client{
		// Default client already follows redirects
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return ""
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return ""
	}

	// Print the JSON response
	//fmt.Println(string(body))

	var dat map[string]interface{}

	if err := json.Unmarshal(body, &dat); err != nil {
		panic(err)
	}

	title := dat["title"].(string)
	au := dat["author"].([]interface{})[0].(map[string]interface{})
	aufam := au["family"].(string)
	aufirst := au["given"].(string)
	year := fmt.Sprint(dat["published-print"].(map[string]interface{})["date-parts"].([]interface{})[0].([]interface{})[0].(float64))

	fmt.Println(title)
	fmt.Println(au)
	fmt.Println(aufam)
	fmt.Println(aufirst)
	fmt.Println(year)

	fmt.Println(mrMultiResponseFromAYT(aufam + ", " + aufirst, year, title))

	return string(body)
}

func argParse(args []string) (string, string, string) {
	//fmt.Println(args)

	author := "";
	year := "";
	title := "";

	if len(args) >= 1 {
		author = args[0]
	}
	if len(args) >= 2 {
		year = args[1];
	}
	if len(args) == 3 {
		title = args[2];
	}

	return author, year, title
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

func main() {
	//p := entryFromDoi("10.1017/fms.2022.74")
	// entryFillMRBib(p)
	//os.Exit(1)
	// Settings for the logger
		
  //m := CreateList("hello", []string{"test 1", "test 2"})

	//p := tea.NewProgram(m)

	//// StartReturningModel returns the model as a tea.Model.
	//mm, err := p.StartReturningModel()
	//if err != nil {
	//	fmt.Println("Oh no:", err)
	//	os.Exit(1)
	//}

	//// Assert the final tea.Model to our local model and print the choice.
	//m, ok := mm.(model)
	//if !ok {
	//	fmt.Println("Oh no")
	//	os.Exit(1)
	//}
	////
	////if m, ok := mm.(model); ok && m.choicee != -1 {
	//fmt.Println(m.choicee)
	////	}

  //os.Exit(1)

	log.SetFlags(log.LstdFlags)

	mrCmd := flag.NewFlagSet("mr", flag.ExitOnError)
  //fooAll := mrCmd.Bool("all", false, "all")
	//fooName := fooCmd.String("name", "", "name")

  doiCmd := flag.NewFlagSet("doi", flag.ExitOnError)
  //doiLevel := barCmd.Int("level", 0, "level")

	arxivCmd := flag.NewFlagSet("arxiv", flag.ExitOnError)

	phdCmd := flag.NewFlagSet("phd", flag.ExitOnError)

	hexhexCmd := flag.NewFlagSet("hexhex", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}

	var res []*entry

	switch os.Args[1] {

	case "hexhex":
		hexhexCmd.Parse(os.Args[2:])
		search := strings.Join(hexhexCmd.Args(), " ")
		res, _ = ZBAnything(search)
		//fmt.Println(result)
	case "mr":
		mrCmd.Parse(os.Args[2:])
		//fmt.Println("subcommand 'mr'")
		//fmt.Println("  all:", *fooAll)
		//fmt.Println("  name:", *fooName)
		//fmt.Println("  tail:", mrCmd.Args())
		if len(mrCmd.Args()) > 3 {
			fmt.Println("expected at most three arguments")
			os.Exit(1)
		}
		author, title, year := argParse(mrCmd.Args())
		//bibtex, _ := mrMultiResponseFromAYT(author, year, title)
		if author == "-" {
			author = ""
		}
		if title == "-" {
			title = ""
		}
		if year == "-" {
			title = ""
		}

		res = MRQueryAYT(author, year, title)
		//fmt.Println(*p)
		//fmt.Println(*p.doi)
	case "doi":
		doiCmd.Parse(os.Args[2:])
		if len(doiCmd.Args()) > 1 {
			fmt.Println("expected one doi")
			os.Exit(1)
		}
		doi, fl := cleanAndValidateDOI(doiCmd.Args()[0])

		if !fl {
			fmt.Println("not a valid doi")
			os.Exit(1)
		}

		res = DOIQuery(doi)
	case "arxiv":
		arxivCmd.Parse(os.Args[2:])
		//fmt.Println("subcommand 'arxiv'")
		//fmt.Println("  all:", *fooAll)
		//fmt.Println("  name:", *fooName)
		if len(arxivCmd.Args()) != 1 {
			fmt.Println("expected exactly one argument")
			os.Exit(1)
		}
		identifier := arxivCmd.Args()[0]
		res = ArxivQueryWithIdentifier(identifier) 
	case "phd":
		phdCmd.Parse(os.Args[2:])
		resp, _ := MGPQueryAndResponse(strings.Join(phdCmd.Args(), " "))
		if len(resp) == 1 {
			res, _ := MGPEntryGetBibtex(resp[0])
			fmt.Print(res)
		}
		if len(resp) > 1 {
			choices := make([]string, len(resp))
			for i := 0; i < len(resp); i++ {
				choices[i] = resp[i].author
				if resp[i].year != "" {
					choices[i] = choices[i] + ", " + resp[i].year
				}
				if resp[i].uni != "" {
					choices[i] = choices[i] + ", " + resp[i].uni
				}
			}
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
				//fmt.Printf("\n---\nYou chose %v!\n", m.choice)
				res, _ := MGPEntryGetBibtex(resp[m.choicee])
				fmt.Print(res)
			}
		}
	//fmt.Println(BibtexEncodeTitle("Hello World, how are you Doing"))
		os.Exit(0)
	default:
		fmt.Println("expected 'mr', 'phd' or 'doi' subcommands")
		os.Exit(1)
	}

	bibtex := res

	if len(bibtex) == 0 {
		fmt.Println("No entry found!")
		os.Exit(0)
	}
	if len(bibtex) > 1 {
		choices := make([]string, len(bibtex))
		for i := 0; i < len(bibtex); i++ {
			choices[i] = bibtex[i].authors[0]
			if len(bibtex[i].authors) > 0 {
			  choices[i] = choices[i] + " et al."
			}
			choices[i] = choices[i] + ", " + bibtex[i].year + ", " + bibtex[i].title
		}
		m := CreateList("Choose entry:", choices)
		p := tea.NewProgram(m)

		// StartReturningModel returns the model as a tea.Model.
		mm, err := p.StartReturningModel()
		if err != nil {
			fmt.Println("Oh no:", err)
			os.Exit(1)
		}
		m, _ = mm.(model)
		if m.choice != "" {
			fmt.Print(bibtex[m.choicee].mrbibtex)
		}
	} else {
		fmt.Print(bibtex[0].mrbibtex)
	}
	os.Exit(0)
}
