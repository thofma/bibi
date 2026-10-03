package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bibi",
	Short: "Retrieve BibTeX for mathematical literature",
	Long: `bibi retrieves BibTeX for DOIs and arXiv identifiers directly, and searches
mathematical literature using MR Lookup, zbMATH Open, Crossref, and the Mathematics
Genealogy Project. It also finds AMS journal abbreviations offline.`,
	SilenceUsage: true,
}

func Execute() {
	if exitCode := commandExitCode(rootCmd); exitCode != 0 {
		os.Exit(exitCode)
	}
}

func commandExitCode(command *cobra.Command) int {
	if err := command.Execute(); err != nil {
		return 1
	}
	return 0
}

func init() {
	addDebugFlag(rootCmd)
	addBibTeXFlags(rootCmd)
}
