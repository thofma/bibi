package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/zb"
	"github.com/thofma/bibi/util"
)

var (
	zbSearch = zb.Search
	zbChoose = util.RunDetailedChooser
)

var zbCmd = &cobra.Command{
	Use:   "zb <search terms...>",
	Short: "Retrieve a BibTeX entry from zbMATH Open",
	Long: `Search zbMATH Open and print a BibTeX entry for a selected result.

When zbMATH returns several results, bibi presents up to ten choices.

Examples:

  bibi zb "zhang p-adic"
  bibi zb "serre local fields"`,
	Args: cobra.MinimumNArgs(1),
	RunE: withDebug(runZB),
}

// hexhexCmd preserves the original prototype command without advertising it.
var hexhexCmd = &cobra.Command{
	Use:    "hexhex <search terms...>",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   withDebug(runZB),
}

func runZB(cmd *cobra.Command, args []string) error {
	if _, err := journalStyle(cmd); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return fmt.Errorf("zbMath search query cannot be empty")
	}

	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching zbMATH Open...")
	response, err := zbSearch(query)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("search zbMATH: %w", err)
	}
	diagnostics.Printf("zbMATH returned %d results", len(response.Result))
	if len(response.Result) == 0 {
		return fmt.Errorf("no zbMath entries found for %q", query)
	}

	results := response.Result
	if len(results) > zb.MaxSearchResults {
		results = results[:zb.MaxSearchResults]
	}

	selected := 0
	if len(results) > 1 {
		choices := make([]util.Choice, len(results))
		for i, result := range results {
			choices[i] = util.Choice{Label: zbChoiceLabel(result), Details: zb.ItemWork(result).Details()}
		}
		selected, err = zbChoose(util.ChooserRequest{Title: "Choose zbMATH result", ChoicePage: util.ChoicePage{Choices: choices},
			Context: cmd.Context(), Output: cmd.ErrOrStderr()})
		if err != nil {
			return fmt.Errorf("choose zbMATH result: %w", err)
		}
		if selected < 0 {
			return fmt.Errorf("zbMath selection cancelled")
		}
		if selected >= len(results) {
			return fmt.Errorf("invalid zbMath selection %d", selected)
		}
	}

	entry, err := zb.ItemToBibEntry(results[selected], response.Result...)
	if err != nil {
		return fmt.Errorf("create BibTeX entry: %w", err)
	}
	diagnostics.Printf("writing zbMATH result %d: %q, key=%s", selected+1, zbChoiceLabel(results[selected]), entry.CiteName)
	return writeBibTeX(cmd, entry, zb.ItemJournalNames(results[selected]))
}

func zbChoiceLabel(item zb.Item) string {
	author := "Unknown author"
	contributors := item.Contributors.Authors
	if len(contributors) == 0 {
		contributors = item.Contributors.Editors
	}
	if len(contributors) > 0 {
		author = contributors[0].Name
		if len(contributors) > 1 {
			author += " et al."
		}
	}

	parts := []string{author}
	if item.Year != "" {
		parts = append(parts, item.Year)
	}
	if title := zb.ItemGetTitle(item); title != "" {
		parts = append(parts, title)
	}
	return strings.Join(parts, ", ")
}

func init() {
	rootCmd.AddCommand(zbCmd)
	rootCmd.AddCommand(hexhexCmd)
}
