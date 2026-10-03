package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/internal/journals"
	"github.com/thofma/bibi/util"
)

func newAbbrCommand(choose func(util.ChooserRequest) (int, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "abbr <journal words...>",
		Short: "Find a journal abbreviation offline",
		Long: `Search the bundled journal abbreviation catalog without network access.
Words can appear in any order and match prefixes in a journal's full title,
translated title or abbreviation. Matching ignores case, accents and punctuation.
A unique result prints immediately; multiple results open an interactive picker.
Only the selected abbreviation is written to standard output.

Examples:

  bibi abbr inventiones mathematicae
  bibi abbr theory number journal
  bibi abbr "reine angewandte"
  bibi abbr "ann. math."`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
	}
	command.RunE = withDebug(func(cmd *cobra.Command, args []string) error {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		query := strings.TrimSpace(strings.Join(args, " "))
		matches, err := journals.Search(query)
		if err != nil {
			return err
		}
		diagnostics.Printf("journal query=%q returned %d results", query, len(matches))
		if len(matches) == 0 {
			return fmt.Errorf("no journal abbreviations found for %q", query)
		}
		selected := 0
		if len(matches) > 1 {
			choices := make([]util.Choice, len(matches))
			for i, journal := range matches {
				choices[i] = journalChoice(journal)
			}
			selected, err = choose(util.ChooserRequest{Title: "Choose journal: " + query,
				ChoicePage: util.ChoicePage{Choices: choices}, Context: cmd.Context(), Output: cmd.ErrOrStderr()})
			if errors.Is(err, util.ErrInteractiveTerminalUnavailable) {
				return fmt.Errorf("%d journals match %q: use a more specific query or rerun in an interactive terminal: %w",
					len(matches), query, util.ErrInteractiveTerminalUnavailable)
			}
			if err != nil {
				return fmt.Errorf("choose journal: %w", err)
			}
			if selected < 0 {
				return errSelectionCancelled
			}
			if selected >= len(matches) {
				return fmt.Errorf("invalid journal selection %d", selected)
			}
		}
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		diagnostics.Printf("writing abbreviation %q", matches[selected].Abbreviation)
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), matches[selected].Abbreviation); err != nil {
			return fmt.Errorf("write journal abbreviation: %w", err)
		}
		return nil
	})
	return command
}

func journalChoice(journal journals.Journal) util.Choice {
	parts := []string{journal.Abbreviation}
	if journal.Title != "" {
		parts = append([]string{journal.Title}, parts...)
	}
	if journal.ISSN != "" {
		parts = append(parts, "ISSN "+journal.ISSN)
	}
	details := "Abbreviation: " + journal.Abbreviation
	if journal.Title != "" {
		details += "\nTitle: " + journal.Title
	}
	if journal.TranslatedTitle != "" {
		details += "\nTranslated title: " + journal.TranslatedTitle
	}
	if journal.ISSN != "" {
		details += "\nISSN: " + journal.ISSN
	}
	return util.Choice{Label: strings.Join(parts, " · "), Details: details}
}

func init() {
	rootCmd.AddCommand(newAbbrCommand(util.RunDetailedChooser))
}
