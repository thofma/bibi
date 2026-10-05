package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/phd"
	"github.com/thofma/bibi/util"
)

var (
	phdQuery     = phd.MGPQueryAndResponseContext
	phdGetBibTeX = phd.MGPEntryGetBibtexContext
	phdChoose    = util.RunDetailedChooser
)

var phdCmd = &cobra.Command{
	Use:   "phd <name...>",
	Short: "Retrieve BibTeX for mathematics PhD theses",
	Long: `Query the Mathematics Genealogy Project for a mathematician's PhD thesis.

Examples:

  bibi phd gauss
  bibi phd "carl gauss"`,
	Args: cobra.MinimumNArgs(1),
	RunE: withDebug(runPhD),
}

func runPhD(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		return fmt.Errorf("a mathematician name is required")
	}

	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching Mathematics Genealogy Project...")
	entries, err := phdQuery(spinner.Context(cmd.Context()), name)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("query Mathematics Genealogy Project: %w", err)
	}
	diagnostics.Printf("Mathematics Genealogy Project returned %d results", len(entries))
	if len(entries) == 0 {
		return fmt.Errorf("no PhD theses found")
	}

	selected := 0
	if len(entries) > 1 {
		choices := make([]util.Choice, len(entries))
		for i, entry := range entries {
			work := bibliography.Work{Title: entry.Title, Authors: []string{entry.Author}, Year: entry.Year, Venue: entry.University,
				Type: "PhD thesis", IDs: map[string]string{"mgp": entry.ID}}
			choices[i] = util.Choice{Label: phdChoiceLabel(entry), Details: work.Details()}
		}
		selected, err = phdChoose(util.ChooserRequest{Title: "Choose thesis", ChoicePage: util.ChoicePage{Choices: choices},
			Context: cmd.Context(), Output: cmd.ErrOrStderr()})
		if err != nil {
			return fmt.Errorf("choose PhD result: %w", err)
		}
		if selected < 0 {
			return fmt.Errorf("PhD selection cancelled")
		}
		if selected >= len(entries) {
			return fmt.Errorf("invalid PhD selection %d", selected)
		}
	}

	spinner = util.StartSpinner(cmd.ErrOrStderr(), "Retrieving Mathematics Genealogy Project thesis...")
	bib, err := phdGetBibTeX(spinner.Context(cmd.Context()), entries[selected])
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("create thesis BibTeX: %w", err)
	}
	if bib == nil {
		return fmt.Errorf("selected PhD result has no BibTeX entry")
	}
	diagnostics.Printf("writing PhD result %d: %q, key=%s", selected+1, phdChoiceLabel(entries[selected]), bib.CiteName)
	return writeBibTeX(cmd, bib)
}

func phdChoiceLabel(entry phd.MGPEntry) string {
	name := strings.TrimSpace(entry.Author)
	if name == "" {
		name = "Unknown author"
	}
	parts := []string{name}
	if year := strings.TrimSpace(entry.Year); year != "" {
		parts = append(parts, year)
	}
	if university := strings.TrimSpace(entry.University); university != "" {
		parts = append(parts, university)
	}
	return strings.Join(parts, ", ")
}

func init() {
	rootCmd.AddCommand(phdCmd)
}
