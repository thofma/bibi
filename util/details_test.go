package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/thofma/bibi/internal/httpclient"
	"github.com/thofma/bibi/lib/bibliography"
)

func detailUpdate(t *testing.T, m detailsModel, msg tea.Msg) (detailsModel, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(detailsModel), cmd
}

func detailKey(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestDetailsFollowHighlightAndScrollWithoutChangingSelection(t *testing.T) {
	m := newDetailsModel(ChooserRequest{ChoicePage: ChoicePage{Choices: []Choice{
		{Label: "First", Details: "Title: First"},
		{Label: "Second", Details: "Title: Second\n\nAuthors: " + strings.Repeat("Full Author Name; ", 100)},
	}}})
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(ansi.Strip(m.details.View()), "Title: Second") || m.list.Index() != 1 {
		t.Fatalf("details did not follow highlight: %s", m.details.View())
	}
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if !strings.Contains(ansi.Strip(m.View()), "Title: Second") {
		t.Fatalf("Tab did not show selected result details: %s", m.View())
	}
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	if m.details.YOffset == 0 || m.list.Index() != 1 {
		t.Fatalf("detail scroll changed selection or failed: offset=%d index=%d", m.details.YOffset, m.list.Index())
	}
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.selected || m.choiceIndex != 1 || m.View() != "" {
		t.Fatalf("selected wrong item from details: %+v", m)
	}
}

func TestChooserSelectionActions(t *testing.T) {
	for _, test := range []struct {
		name, keys string
		enabled    bool
		selected   int
		calls      int
	}{
		{"alternate selection", "m", true, 0, 1},
		{"alternate selection from details", "\x1b[B\tm", true, 1, 1},
		{"Enter keeps default action", "\r", true, 0, 0},
		{"cancel skips action", "q", true, -1, 0},
		{"disabled key is ignored", "m\r", false, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			calls := 0
			request := ChooserRequest{Context: ctx, Output: io.Discard,
				ChoicePage: ChoicePage{Choices: []Choice{{Label: "First"}, {Label: "Second"}}}}
			if test.enabled {
				request.Actions = []ChooserAction{{Key: "m", Label: "MR BibTeX", OnSelect: func() { calls++ }}}
			}
			input := &trackedChooserInput{Reader: strings.NewReader(test.keys)}
			selected, err := runDetailedChooser(request, func() (io.ReadCloser, error) { return input, nil })
			if err != nil || selected != test.selected || calls != test.calls || !input.closed {
				t.Fatalf("selection=%d calls=%d closed=%t error=%v", selected, calls, input.closed, err)
			}
		})
	}
}

func TestDetailsSelectionActionUsesLaterPageHighlight(t *testing.T) {
	calls := 0
	m := newDetailsModel(ChooserRequest{
		ChoicePage: ChoicePage{Choices: []Choice{{Label: "First"}, {Label: "Second"}}, NextToken: "next"},
		Actions:    []ChooserAction{{Key: "m", Label: "MR BibTeX", OnSelect: func() { calls++ }}},
		LoadPage: func(context.Context, string) (ChoicePage, error) {
			return ChoicePage{Choices: []Choice{{Label: "Third"}, {Label: "Fourth"}}}, nil
		},
	})
	m, load := detailUpdate(t, m, detailKey("n"))
	blocked, command := detailUpdate(t, m, detailKey("m"))
	if blocked.selected || command != nil || strings.Contains(ansi.Strip(blocked.View()), "MR BibTeX") {
		t.Fatal("alternate selection was available while loading")
	}
	m, _ = detailUpdate(t, m, load())
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m, command = detailUpdate(t, m, detailKey("m"))
	if !m.selected || m.choiceIndex != 3 || command == nil || calls != 0 {
		t.Fatalf("selection=%d selected=%t calls=%d command present=%t", m.choiceIndex, m.selected, calls, command != nil)
	}
}

func TestDetailsSelectionActionHintFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 30}, {80, 24}, {40, 12}, {24, 10}, {16, 8}} {
		for _, details := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d/details=%t", size[0], size[1], details), func(t *testing.T) {
				m := newDetailsModel(ChooserRequest{
					ChoicePage: ChoicePage{Choices: []Choice{{Label: "First", Details: "Title: First"}}, NextToken: "next"},
					Actions:    []ChooserAction{{Key: "m", Label: "MR BibTeX"}},
					LoadPage:   func(context.Context, string) (ChoicePage, error) { return ChoicePage{}, nil },
				})
				m, _ = detailUpdate(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				if details {
					m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
				}
				view := ansi.Strip(m.View())
				if !strings.Contains(view, "MR BibTeX") || lipgloss.Height(view) > size[1] {
					t.Fatalf("missing shortcut or overflowing height:\n%s", view)
				}
				for _, line := range strings.Split(view, "\n") {
					if ansi.StringWidth(line) > size[0] {
						t.Fatalf("line overflows: %q", line)
					}
				}
			})
		}
	}
}

func TestDetailsPagingCachesPagesAndKeepsGlobalIndices(t *testing.T) {
	loads := 0
	m := newDetailsModel(ChooserRequest{ChoicePage: ChoicePage{Choices: []Choice{{Label: "First"}, {Label: "Second"}}, NextToken: "opaque", Total: 4},
		LoadPage: func(_ context.Context, token string) (ChoicePage, error) {
			loads++
			if token != "opaque" {
				t.Fatalf("token = %q", token)
			}
			return ChoicePage{Choices: []Choice{{Label: "Third"}, {Label: "Fourth"}}, Total: 4}, nil
		}})
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := detailUpdate(t, m, detailKey("n"))
	if !m.loading || cmd == nil {
		t.Fatal("next page was not requested")
	}
	event := httpclient.Event{Service: "zbMATH Open", Attempt: 2, Delay: time.Second, Reason: "HTTP 503 Service Unavailable"}
	m, _ = detailUpdate(t, m, retryProgressMsg{event})
	if !strings.Contains(ansi.Strip(m.View()), "retrying in 1s") || m.pageIndex != 0 || m.list.Index() != 1 {
		t.Fatalf("retry progress lost the current page or selection: %s", m.View())
	}
	blocked, _ := detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if blocked.selected {
		t.Fatal("selected while loading")
	}
	m, _ = detailUpdate(t, m, cmd())
	if m.pageIndex != 1 || !strings.Contains(m.View(), "Results 3–4") || strings.Contains(m.View(), "retrying") {
		t.Fatalf("wrong second page: %s", m.View())
	}
	m, _ = detailUpdate(t, m, detailKey("p"))
	if m.pageIndex != 0 || m.list.Index() != 1 {
		t.Fatal("previous page did not restore highlight")
	}
	m, cmd = detailUpdate(t, m, detailKey("n"))
	if cmd != nil || loads != 1 {
		t.Fatal("cached page requested again")
	}
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.choiceIndex != 3 {
		t.Fatalf("global selection = %d, want 3", m.choiceIndex)
	}
}

func TestDetailsPageFailureRetryAndEmptyEnd(t *testing.T) {
	calls := 0
	m := newDetailsModel(ChooserRequest{ChoicePage: ChoicePage{Choices: []Choice{{Label: "Keep this result"}}, NextToken: "same-token"},
		LoadPage: func(_ context.Context, token string) (ChoicePage, error) {
			calls++
			if token != "same-token" {
				t.Fatal("retry advanced continuation token")
			}
			if calls == 1 {
				return ChoicePage{}, errors.New("service unavailable")
			}
			return ChoicePage{}, nil
		}})
	m, cmd := detailUpdate(t, m, detailKey("n"))
	m, _ = detailUpdate(t, m, cmd())
	if m.pageIndex != 0 || m.list.Index() != 0 || !strings.Contains(m.View(), "retry") || !strings.Contains(m.View(), "Keep this result") {
		t.Fatalf("failed page lost current results: %s", m.View())
	}
	m, cmd = detailUpdate(t, m, detailKey("n"))
	m, _ = detailUpdate(t, m, cmd())
	if m.nextToken != "" || len(m.pages) != 1 || m.loadError != nil {
		t.Fatal("empty terminal page was presented as a result page")
	}
	m, cmd = detailUpdate(t, m, detailKey("n"))
	if cmd != nil || calls != 2 {
		t.Fatal("requested a page after the end")
	}
	m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.selected || m.choiceIndex != 0 {
		t.Fatal("could not select original result after failed and empty requests")
	}
}

func TestDetailsCancellationWhileLoading(t *testing.T) {
	m := newDetailsModel(ChooserRequest{ChoicePage: ChoicePage{Choices: []Choice{{Label: "First"}}, NextToken: "next"},
		LoadPage: func(context.Context, string) (ChoicePage, error) { return ChoicePage{}, nil }})
	m, _ = detailUpdate(t, m, detailKey("n"))
	m, cmd := detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.quitting || m.selected || cmd == nil || m.View() != "" {
		t.Fatal("loading prevented cancellation")
	}
}

func TestDetailsResizeLongMetadataAndTerminalEscapes(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, size := range [][2]int{{120, 30}, {80, 24}, {40, 12}, {60, 18}, {24, 10}, {16, 8}} {
			t.Run(fmt.Sprintf("%dx%d/dark=%v", size[0], size[1], dark), func(t *testing.T) {
				m := newDetailsModel(ChooserRequest{Title: "Search\nremote\x1b[2J title", ChoicePage: ChoicePage{Choices: []Choice{
					{Label: strings.Repeat("Long 数学 title ", 40), Details: "Title: " + strings.Repeat("Long 数学 title ", 40) +
						"\n\nDOI: 10.1000/" + strings.Repeat("unbroken", 60) + "\x1b]52;c;YWJj\a"},
				}}})
				renderer := lipgloss.NewRenderer(io.Discard)
				renderer.SetColorProfile(termenv.TrueColor)
				renderer.SetHasDarkBackground(dark)
				m.theme = chooserThemeFromRenderer(renderer)
				m.list.SetDelegate(detailDelegate{theme: m.theme})
				m, _ = detailUpdate(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
				view := ansi.Strip(m.View())
				if lipgloss.Height(view) > size[1] {
					t.Errorf("%v: height %d exceeds terminal", size, lipgloss.Height(view))
				}
				for _, line := range strings.Split(view, "\n") {
					if ansi.StringWidth(line) > size[0] {
						t.Errorf("%v: line overflows: %q", size, line)
					}
				}
				if !strings.Contains(view, "Title:") || strings.Contains(view, "YWJj") || strings.Contains(view, "\x1b") {
					t.Errorf("%v: metadata missing or unsafe escapes: %q", size, view)
				}
			})
		}
	}
}

func TestDetailsResultsFillTerminalHeight(t *testing.T) {
	choices := make([]Choice, 100)
	for i := range choices {
		choices[i] = Choice{Label: fmt.Sprintf("Result%03d", i+1), Details: "Title: Selected citation\n" + strings.Repeat("Metadata line\n", 60)}
	}
	for _, size := range [][2]int{{80, 24}, {80, 48}, {120, 48}, {40, 32}, {24, 10}, {16, 8}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := newDetailsModel(ChooserRequest{ChoicePage: ChoicePage{Choices: choices}})
			m, _ = detailUpdate(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			view := ansi.Strip(m.View())
			if got := lipgloss.Height(view); got != size[1] {
				t.Fatalf("selector height = %d, want %d:\n%s", got, size[1], view)
			}
			if visible := strings.Count(view, "Result0"); size[1] >= 24 && visible <= 10 {
				t.Errorf("only %d results visible in a %d-row terminal:\n%s", visible, size[1], view)
			}
			m.list.Select(65)
			m.refreshDetails(true)
			m, _ = detailUpdate(t, m, tea.WindowSizeMsg{Width: size[0], Height: max(8, size[1]-4)})
			if m.list.Index() != 65 || !strings.Contains(ansi.Strip(m.View()), "Result066") {
				t.Fatalf("resize lost the highlighted result: index=%d\n%s", m.list.Index(), m.View())
			}
			m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
			if !strings.Contains(ansi.Strip(m.View()), "Title:") || m.list.Index() != 65 {
				t.Fatalf("Tab did not show details for the selected result:\n%s", m.View())
			}
			m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyTab})
			if !strings.Contains(ansi.Strip(m.View()), "Result066") || m.list.Index() != 65 {
				t.Fatalf("Tab did not restore the highlighted result:\n%s", m.View())
			}
		})
	}
}

func TestChooserPageSizeMatchesDefaultVisibleResults(t *testing.T) {
	choices := make([]Choice, 100)
	for i := range choices {
		choices[i] = Choice{Label: fmt.Sprintf("Result%03d", i+1)}
	}
	m := newDetailsModel(ChooserRequest{Output: io.Discard, ChoicePage: ChoicePage{Choices: choices}})
	visible := strings.Count(ansi.Strip(m.View()), "Result0")
	if pageSize := ChooserPageSize(io.Discard); pageSize != visible || pageSize <= 10 {
		t.Fatalf("page size=%d visible results=%d", pageSize, visible)
	}
}

func TestChooserMissingTerminalReportsActionableErrorWithoutRendering(t *testing.T) {
	var output strings.Builder
	request := ChooserRequest{Title: "Confirm mr BibTeX", Output: &output, ChoicePage: ChoicePage{Choices: []Choice{{Label: "Identity unverified"}}}}
	selected, err := runDetailedChooser(request, func() (io.ReadCloser, error) { return nil, errors.New("no controlling terminal") })
	if selected != -1 || !errors.Is(err, ErrInteractiveTerminalUnavailable) || !strings.Contains(err.Error(), "rerun in an interactive terminal") || output.Len() != 0 {
		t.Fatalf("selection=%d error=%v output=%q", selected, err, output.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request.Context = ctx
	_, err = runDetailedChooser(request, func() (io.ReadCloser, error) {
		t.Fatal("cancelled chooser opened terminal input")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled chooser error=%v", err)
	}
}

type trackedChooserInput struct {
	io.Reader
	closed bool
}

func (input *trackedChooserInput) Close() error {
	input.closed = true
	return nil
}

func TestChooserConfirmationDoesNotReadPipedStdin(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	const identifiers = "10.1000/first\n10.1000/last\n"
	if _, err := io.WriteString(writer, identifiers); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	originalStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = originalStdin })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	input := &trackedChooserInput{Reader: strings.NewReader("\r")}
	selected, err := runDetailedChooser(ChooserRequest{Title: "Confirm mr BibTeX", Context: ctx, Output: io.Discard,
		ChoicePage: ChoicePage{Choices: []Choice{{Label: "Identity unverified", Details: "Review this entry"}}}},
		func() (io.ReadCloser, error) { return input, nil })
	remaining, readErr := io.ReadAll(reader)
	if err != nil || selected != 0 || !input.closed || readErr != nil || string(remaining) != identifiers {
		t.Fatalf("selection=%d error=%v closed=%t remaining=%q read error=%v", selected, err, input.closed, remaining, readErr)
	}
}

func TestComparisonRemainsReadableWithoutColorInCompactDetailsPane(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	selected := bibliography.Work{Title: "Selected work", Edition: "1"}
	candidate := bibliography.Work{Title: "Provider work", Edition: "2", Notes: "Translation of the original"}
	details := bibliography.ComparisonDetails(selected, candidate, bibliography.AssessMatch(selected, candidate, "mr"))
	m := newDetailsModel(ChooserRequest{Title: "Confirm mr BibTeX", Confirmation: true, ChoicePage: ChoicePage{Choices: []Choice{{Label: "Identity unverified", Details: details}}}})
	m, _ = detailUpdate(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	view := m.View()
	if strings.Contains(view, "\x1b") || !strings.Contains(view, "Identity unverified") || !m.detailsFocus {
		t.Fatalf("match status is unavailable without color: %q", view)
	}
	for i := 0; i < 30; i++ {
		m, _ = detailUpdate(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if !strings.Contains(m.View(), "Translation of the original") || m.selected {
		t.Fatalf("comparison cannot be reviewed independently of selection: %s", m.View())
	}
}
