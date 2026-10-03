package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/mr"
	"github.com/thofma/bibi/util"
)

var (
	mrQuery  = mr.MRQueryAYTContext
	mrChoose = util.RunDetailedChooser
)

var mrCmd = &cobra.Command{
	Use:   "mr <author> [title] [year]",
	Short: "Retrieve BibTeX from the MR Lookup service",
	Long: `Query the free MR Lookup service from the American Mathematical Society.

A field can be omitted with "-"; title and year may also be left off.

Examples:

  bibi mr serre "a course in arithmetic" 1973
  bibi mr serre - 1973`,
	Args: cobra.RangeArgs(1, 3),
	RunE: withDebug(runMR),
}

func runMR(cmd *cobra.Command, args []string) error {
	if _, err := journalStyle(cmd); err != nil {
		return err
	}
	if len(args) < 1 || len(args) > 3 {
		return fmt.Errorf("expected one to three MR search arguments")
	}

	author := dashToEmpty(args[0])
	title := ""
	year := ""
	if len(args) >= 2 {
		title = dashToEmpty(args[1])
	}
	if len(args) == 3 {
		year = dashToEmpty(args[2])
	}

	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching MR Lookup...")
	entries, err := mrQuery(spinner.Context(cmd.Context()), author, year, title)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("query MR Lookup: %w", err)
	}
	diagnostics.Printf("MR Lookup returned %d results", len(entries))
	if len(entries) == 0 {
		return fmt.Errorf("no MR entries found")
	}

	selected := 0
	if len(entries) > 1 {
		choices := make([]util.Choice, len(entries))
		for i, entry := range entries {
			choices[i].Label = mrChoiceLabel(entry)
			if entry != nil {
				work := bibliography.WorkFromEntry(entry.BibTeX)
				work.Title, work.Authors, work.Year = entry.Title, entry.Authors, entry.Year
				if entry.BibTeX != nil {
					work.IDs = map[string]string{"mr": entry.BibTeX.CiteName}
				}
				choices[i].Details = work.Details()
			}
		}
		selected, err = mrChoose(util.ChooserRequest{Title: "Choose MR result", ChoicePage: util.ChoicePage{Choices: choices},
			Context: cmd.Context(), Output: cmd.ErrOrStderr()})
		if err != nil {
			return fmt.Errorf("choose MR result: %w", err)
		}
		if selected < 0 {
			return fmt.Errorf("MR selection cancelled")
		}
		if selected >= len(entries) {
			return fmt.Errorf("invalid MR selection %d", selected)
		}
	}

	entry := entries[selected]
	if entry == nil || entry.BibTeX == nil {
		return fmt.Errorf("MR result %d has no BibTeX entry", selected+1)
	}
	diagnostics.Printf("writing MR result %d: %q, key=%s", selected+1, mrChoiceLabel(entry), entry.BibTeX.CiteName)
	return writeBibTeX(cmd, entry.BibTeX, bibliography.JournalNamesFromEntry(entry.BibTeX))
}

func mrChoiceLabel(entry *mr.Entry) string {
	if entry == nil {
		return "Unknown author"
	}

	author := "Unknown author"
	if len(entry.Authors) > 0 {
		if firstAuthor := strings.TrimSpace(entry.Authors[0]); firstAuthor != "" {
			author = firstAuthor
			if len(entry.Authors) > 1 {
				author += " et al."
			}
		}
	}

	parts := []string{author}
	if entry.Year != "" {
		parts = append(parts, entry.Year)
	}
	if entry.Title != "" {
		parts = append(parts, entry.Title)
	}
	return strings.Join(parts, ", ")
}

func dashToEmpty(value string) string {
	if value == "-" {
		return ""
	}
	return value
}

func init() {
	rootCmd.AddCommand(mrCmd)
}
