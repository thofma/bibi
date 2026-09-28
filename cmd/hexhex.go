package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/zb"
	"github.com/thofma/bibi/util"
)

var zbSearch = zb.Search
var zbChoose = util.RunChooser

var zbCmd = &cobra.Command{
	Use:   "zb <search terms>",
	Short: "Retrieve a BibTeX entry from zbMATH Open",
	Long: `Search zbMATH Open and print a BibTeX entry for a selected result.

When zbMATH returns several results, bibi presents up to ten choices.

Examples:

  bibi zb "hofmann zhang p-adic"
  bibi zb "serre local fields"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runZB,
}

// hexhexCmd preserves the original prototype command without advertising it.
var hexhexCmd = &cobra.Command{
	Use:    "hexhex <search terms>",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   runZB,
}

func runZB(cmd *cobra.Command, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return fmt.Errorf("zbMath search query cannot be empty")
	}

	response, err := zbSearch(query)
	if err != nil {
		return err
	}
	if len(response.Result) == 0 {
		return fmt.Errorf("no zbMath entries found for %q", query)
	}

	results := response.Result
	if len(results) > zb.MaxSearchResults {
		results = results[:zb.MaxSearchResults]
	}

	selected := 0
	if len(results) > 1 {
		choices := make([]string, len(results))
		for i, result := range results {
			choices[i] = zbChoiceLabel(result)
		}
		selected = zbChoose(choices)
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
	if _, err := fmt.Fprint(cmd.OutOrStdout(), entry.PrettyString()); err != nil {
		return fmt.Errorf("write BibTeX entry: %w", err)
	}
	return nil
}

func zbChoiceLabel(item zb.Item) string {
	author := "Unknown author"
	if len(item.Contributors.Authors) > 0 {
		author = item.Contributors.Authors[0].Name
		if len(item.Contributors.Authors) > 1 {
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
