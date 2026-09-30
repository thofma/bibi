package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/phd"
	"github.com/thofma/bibi/util"
)

var (
	phdQuery     = phd.MGPQueryAndResponse
	phdGetBibTeX = phd.MGPEntryGetBibtex
	phdChoose    = util.RunChooser
)

var phdCmd = &cobra.Command{
	Use:   "phd <name...>",
	Short: "Retrieve BibTeX for mathematics PhD theses",
	Long: `Query the Mathematics Genealogy Project for a mathematician's PhD thesis.

Examples:

  bibi phd gauss
  bibi phd "carl gauss"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPhD,
}

func runPhD(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		return fmt.Errorf("a mathematician name is required")
	}

	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching Mathematics Genealogy Project...")
	entries, err := phdQuery(name)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("query Mathematics Genealogy Project: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no PhD theses found")
	}

	selected := 0
	if len(entries) > 1 {
		choices := make([]string, len(entries))
		for i, entry := range entries {
			choices[i] = phdChoiceLabel(entry)
		}
		selected, err = phdChoose(choices)
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

	bib, err := phdGetBibTeX(entries[selected])
	if err != nil {
		return fmt.Errorf("create thesis BibTeX: %w", err)
	}
	if bib == nil {
		return fmt.Errorf("selected PhD result has no BibTeX entry")
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), formatBibTeX(bib)); err != nil {
		return fmt.Errorf("write BibTeX entry: %w", err)
	}
	return nil
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
