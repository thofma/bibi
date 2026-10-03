package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/thofma/bibi/util"
)

func TestAbbrUniqueMatches(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"inventiones", "mathematicae"}, "Invent. Math.\n"},
		{[]string{"MATHÉMATÍCAE, INVENTIONES"}, "Invent. Math.\n"},
		{[]string{"reine angewandte"}, "J. Reine Angew. Math.\n"},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			root := debugTestRoot(newAbbrCommand(func(util.ChooserRequest) (int, error) {
				t.Fatal("unique match opened a picker")
				return -1, nil
			}))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(append([]string{"abbr"}, test.args...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != test.want || stderr.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestAbbrMultipleMatchesShowJournalMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := context.WithValue(context.Background(), struct{}{}, "fixture context")
	root := debugTestRoot(newAbbrCommand(func(request util.ChooserRequest) (int, error) {
		if request.Context == nil || request.Context.Value(struct{}{}) != ctx.Value(struct{}{}) || request.Output != &stderr || request.Confirmation || len(request.Choices) <= 1 {
			t.Fatalf("unexpected chooser request: %+v", request)
		}
		if !strings.Contains(request.Title, "theor numbe jour") {
			t.Errorf("picker title = %q", request.Title)
		}
		for i, choice := range request.Choices {
			if strings.Contains(choice.Details, "Abbreviation: J. Number Theory\n") {
				for _, value := range []string{"Journal of Number Theory", "J. Number Theory", "ISSN 0022-314X"} {
					if !strings.Contains(choice.Label, value) {
						t.Errorf("choice label %q is missing %q", choice.Label, value)
					}
				}
				if !strings.Contains(choice.Details, "ISSN: 0022-314X") {
					t.Errorf("choice details = %q", choice.Details)
				}
				_, _ = fmt.Fprintln(request.Output, "picker output")
				return i, nil
			}
		}
		t.Fatal("Journal of Number Theory not offered")
		return -1, nil
	}))
	root.SetContext(ctx)
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"abbr", "theor", "numbe", "jour"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "J. Number Theory\n" || stderr.String() != "picker output\n" {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestAbbrFailuresLeaveStdoutEmpty(t *testing.T) {
	chooserError := errors.New("picker failed")
	for _, test := range []struct {
		name, want string
		args       []string
		selection  int
		chooseErr  error
		wantErr    error
	}{
		{name: "missing query", args: nil, want: "requires at least 1"},
		{name: "blank query", args: []string{" \t "}, want: "at least one letter or number"},
		{name: "punctuation query", args: []string{"..."}, want: "at least one letter or number"},
		{name: "no matches", args: []string{"journalwhichdoesnotexist"}, want: "no journal abbreviations"},
		{name: "cancelled picker", args: []string{"number theory"}, selection: -1, wantErr: errSelectionCancelled},
		{name: "invalid selection", args: []string{"number theory"}, selection: 100000, want: "invalid journal selection"},
		{name: "picker failure", args: []string{"number theory"}, chooseErr: chooserError, wantErr: chooserError},
		{name: "no terminal", args: []string{"number theory"}, chooseErr: util.ErrInteractiveTerminalUnavailable,
			want: "use a more specific query", wantErr: util.ErrInteractiveTerminalUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := debugTestRoot(newAbbrCommand(func(util.ChooserRequest) (int, error) {
				return test.selection, test.chooseErr
			}))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(append([]string{"abbr"}, test.args...))
			err := root.Execute()
			if err == nil || stdout.Len() != 0 {
				t.Fatalf("error=%v stdout=%q", err, stdout.String())
			}
			if test.want != "" && !strings.Contains(err.Error(), test.want) {
				t.Errorf("error=%v, want %q", err, test.want)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Errorf("error=%v, want wrapped %v", err, test.wantErr)
			}
		})
	}
}

func TestAbbrContextCancellationPreventsOutput(t *testing.T) {
	for _, query := range []string{"inventiones mathematicae", "number theory"} {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		root := debugTestRoot(newAbbrCommand(func(util.ChooserRequest) (int, error) {
			cancel()
			return 0, nil
		}))
		if query == "inventiones mathematicae" {
			cancel()
		}
		var stdout bytes.Buffer
		root.SetContext(ctx)
		root.SetOut(&stdout)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"abbr", query})
		if err := root.Execute(); !errors.Is(err, context.Canceled) || stdout.Len() != 0 {
			t.Errorf("query=%q error=%v stdout=%q", query, err, stdout.String())
		}
	}
}

func TestAbbrDebugKeepsAbbreviationOnStdout(t *testing.T) {
	root := debugTestRoot(newAbbrCommand(nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"abbr", "inventiones mathematicae", "--debug"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Invent. Math.\n" || !strings.Contains(stderr.String(), "journal query=") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestAbbrIsRegistered(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"abbr"})
	if err != nil || command == rootCmd || command.Name() != "abbr" || command.RunE == nil {
		t.Fatalf("abbr is not registered: command=%v error=%v", command, err)
	}
}
