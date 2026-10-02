package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/bibfile"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/util"
)

func newAddCommand(services func() searchServices, choose func(util.ChooserRequest) (int, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "add <query> <file>",
		Short: "Find a work and add its BibTeX to a bibliography file",
		Long: `Find a work using the same discovery and BibTeX providers as search, then
append its entry to a bibliography file. Quote multiword queries.

Existing text is preserved. A shared DOI or database identifier skips a duplicate;
a citation-key collision or conflicting identifiers stops the add.

Examples:
  bibi add "serre local fields" references.bib --bib mr
  bibi add "10.1016/j.jnt.2016.05.016" references.bib --key Hofmann2016
  bibi add "serre local fields" references.bib --dry-run`,
		SilenceUsage: true,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("add requires exactly two arguments: <query> <file>; quote multiword queries, e.g. bibi add \"local fields\" references.bib")
			}
			if strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[1]) == "" {
				return fmt.Errorf("query and bibliography file cannot be empty")
			}
			return nil
		},
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 1 {
				return []string{"bib"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	addSearchFlags(command)
	command.Flags().String("key", "", "Citation key for the new entry (default: provider's key)")
	command.Flags().Bool("dry-run", false, "Print the proposed entry without changing or creating the file")
	command.RunE = withDebug(func(cmd *cobra.Command, args []string) error {
		if _, err := journalStyle(cmd); err != nil {
			return err
		}
		key, _ := cmd.Flags().GetString("key")
		if cmd.Flags().Changed("key") {
			if err := bibfile.ValidateKey(key); err != nil {
				return err
			}
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		path := args[1]
		diagnostics.Printf("add file=%q dry_run=%t key_override=%q stage=preflight", path, dryRun, key)
		if err := bibfile.ValidateTarget(path, !dryRun); err != nil {
			return fmt.Errorf("check bibliography %q: %w", path, err)
		}
		record, err := retrieveCitation(cmd, strings.TrimSpace(args[0]), services, choose)
		if err != nil {
			return err
		}
		input := *record.Entry
		if cmd.Flags().Changed("key") {
			input.CiteName = key
		}
		if err := bibfile.ValidateKey(input.CiteName); err != nil {
			return err
		}
		entry, err := prepareBibTeX(cmd, &input, record.Journals)
		if err != nil {
			return err
		}
		diagnostics.Printf("add file=%q key=%q stage=save started", path, entry.CiteName)
		result, err := bibfile.Add(cmd.Context(), path, formatBibTeX(entry), dryRun)
		if err != nil {
			diagnostics.Printf("add file=%q stage=save failed: %v; provider lookup and matching succeeded", path, err)
			return fmt.Errorf("add to %q: %w", path, err)
		}
		if len(result.ExistingKeys) > 0 {
			diagnostics.Printf("add file=%q stage=duplicate existing_keys=%q (verified exported identifiers)", path, result.ExistingKeys)
			_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Already present in %q as %s\n", path, strings.Join(result.ExistingKeys, ", "))
			return err
		}
		if dryRun {
			if _, err := fmt.Fprint(cmd.OutOrStdout(), result.Entry); err != nil {
				return fmt.Errorf("write proposed BibTeX entry: %w", err)
			}
			_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Would add %s to %q\n", result.Key, path)
			return err
		}
		diagnostics.Printf("add file=%q stage=save succeeded: key=%q created=%t", path, result.Key, result.Created)
		_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Added %s to %q\n", result.Key, path)
		return err
	})
	return command
}

func init() {
	rootCmd.AddCommand(newAddCommand(defaultSearchServices, util.RunDetailedChooser))
}
