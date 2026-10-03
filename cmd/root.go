package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/buildinfo"
)

var rootCmd = &cobra.Command{
	Use:   "bibi",
	Short: "Retrieve BibTeX for mathematical literature",
	Long: `bibi retrieves BibTeX for DOIs and arXiv identifiers directly, and searches
mathematical literature using MR Lookup, zbMATH Open, Crossref, and the Mathematics
Genealogy Project. It also finds journal abbreviations offline.`,
	SilenceUsage: true,
	Version:      buildinfo.String(),
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	rootCmd.SetContext(ctx)
	defer stop()
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
	rootCmd.SetVersionTemplate("bibi {{.Version}}\n")
	addDebugFlag(rootCmd)
	addBibTeXFlags(rootCmd)
}
