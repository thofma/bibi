package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/zb"
)

func TestRootCommandConfiguration(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Error("root command should not print usage after runtime errors")
	}
	if flag := rootCmd.Flags().Lookup("toggle"); flag != nil {
		t.Errorf("unexpected generated toggle flag: %#v", flag)
	}
}

func TestLookupCommandsUseRunE(t *testing.T) {
	for _, command := range []struct {
		name string
		runE func(*cobra.Command, []string) error
		run  func(*cobra.Command, []string)
	}{
		{name: "zb", runE: zbCmd.RunE, run: zbCmd.Run},
		{name: "hexhex", runE: hexhexCmd.RunE, run: hexhexCmd.Run},
		{name: "mr", runE: mrCmd.RunE, run: mrCmd.Run},
		{name: "phd", runE: phdCmd.RunE, run: phdCmd.Run},
	} {
		if command.runE == nil {
			t.Errorf("%s command has no RunE", command.name)
		}
		if command.run != nil {
			t.Errorf("%s command still uses Run", command.name)
		}
	}
}

func TestRootCommandExitCodes(t *testing.T) {
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	t.Run("successful lookup", func(t *testing.T) {
		useZBSearch(t, func(string) (zb.Response, error) {
			return zb.Response{Result: []zb.Item{journalResult(1, "Fixture result")}}, nil
		})

		var stdout, stderr bytes.Buffer
		rootCmd.SetOut(&stdout)
		rootCmd.SetErr(&stderr)
		rootCmd.SetArgs([]string{"zb", "fixture"})
		if got, want := commandExitCode(rootCmd), 0; got != want {
			t.Fatalf("exit code = %d, want %d; stderr: %s", got, want, stderr.String())
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
		parsed, err := bibtex.Parse(strings.NewReader(stdout.String()))
		if err != nil {
			t.Fatalf("stdout is not valid BibTeX: %v\n%s", err, stdout.String())
		}
		if got, want := len(parsed.Entries), 1; got != want {
			t.Errorf("BibTeX entries = %d, want %d", got, want)
		}
	})

	t.Run("invalid arguments", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		rootCmd.SetOut(&stdout)
		rootCmd.SetErr(&stderr)
		rootCmd.SetArgs([]string{"mr"})
		if got, want := commandExitCode(rootCmd), 1; got != want {
			t.Fatalf("exit code = %d, want %d", got, want)
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout = %q, want no BibTeX on error", stdout.String())
		}
		if !strings.Contains(stderr.String(), "accepts between 1 and 3") {
			t.Errorf("stderr = %q, want argument error", stderr.String())
		}
		if strings.Contains(stderr.String(), "Usage:") {
			t.Errorf("stderr unexpectedly contains usage: %q", stderr.String())
		}
	})
}
