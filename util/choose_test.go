package util

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunChooserRejectsEmptyChoices(t *testing.T) {
	if _, err := RunChooser(nil); err == nil || !strings.Contains(err.Error(), "at least one option") {
		t.Fatalf("RunChooser() error = %v, want empty-choice error", err)
	}
}

func TestChooserSelection(t *testing.T) {
	t.Run("selected", func(t *testing.T) {
		choice, err := chooserSelection(model{choiceIndex: 2, selected: true})
		if err != nil {
			t.Fatalf("chooserSelection() error = %v", err)
		}
		if got, want := choice, 2; got != want {
			t.Errorf("choice = %d, want %d", got, want)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		choice, err := chooserSelection(model{quitting: true})
		if err != nil {
			t.Fatalf("chooserSelection() error = %v", err)
		}
		if got, want := choice, -1; got != want {
			t.Errorf("choice = %d, want %d", got, want)
		}
	})

	t.Run("unexpected model", func(t *testing.T) {
		if _, err := chooserSelection(nil); err == nil || !strings.Contains(err.Error(), "unexpected model") {
			t.Fatalf("chooserSelection() error = %v, want model-type error", err)
		}
	})
}

func TestChooserModelRecordsSelectionAndCancellation(t *testing.T) {
	t.Run("selection", func(t *testing.T) {
		initial := CreateList("Choose result", []string{"first", "second"})
		updated, _ := initial.Update(tea.KeyMsg{Type: tea.KeyEnter})
		result, ok := updated.(model)
		if !ok {
			t.Fatalf("updated model = %T, want util.model", updated)
		}
		if !result.selected || result.choiceIndex != 0 {
			t.Errorf("model = %#v, want selected first item", result)
		}
		if got := result.View(); got != "" {
			t.Errorf("selected View() = %q, want empty", got)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		initial := CreateList("Choose result", []string{"first"})
		updated, _ := initial.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		result, ok := updated.(model)
		if !ok {
			t.Fatalf("updated model = %T, want util.model", updated)
		}
		if !result.quitting {
			t.Errorf("model = %#v, want quitting state", result)
		}
	})
}
