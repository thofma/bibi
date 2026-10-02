package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

func addBibTeXFlags(command *cobra.Command) {
	command.PersistentFlags().String("journal", "source", "Journal name: source, short, full")
}

func journalStyle(command *cobra.Command) (string, error) {
	style := "source"
	if command.Flags().Lookup("journal") != nil {
		style, _ = command.Flags().GetString("journal")
	}
	switch style {
	case "source", "short", "full":
		return style, nil
	default:
		return "", fmt.Errorf("unknown journal style %q (choose source, short, or full)", style)
	}
}

// writeBibTeX is the shared output path for all providers. Warnings never enter
// the BibTeX stream or prompt for another selection.
func writeBibTeX(command *cobra.Command, entry *bibtex.BibEntry, names bibliography.JournalNames) error {
	entry, err := prepareBibTeX(command, entry, names)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(command.OutOrStdout(), formatBibTeX(entry)); err != nil {
		return fmt.Errorf("write BibTeX entry: %w", err)
	}
	return nil
}

// prepareBibTeX applies the shared journal preference and quality warnings once.
func prepareBibTeX(command *cobra.Command, entry *bibtex.BibEntry, names bibliography.JournalNames) (*bibtex.BibEntry, error) {
	style, err := journalStyle(command)
	if err != nil {
		return nil, err
	}
	entry, warning := bibliography.WithJournalName(entry, names, style)
	var warnings []string
	if warning != "" {
		warnings = append(warnings, warning)
	}
	if missing := bibliography.MissingFields(entry); len(missing) > 0 {
		warnings = append(warnings, "missing required BibTeX fields: "+strings.Join(missing, ", "))
	}
	for _, warning := range warnings {
		diagnostics.Printf("bib key=%q stage=quality warning: %s", entry.CiteName, warning)
		if _, err := fmt.Fprintf(command.ErrOrStderr(), "Warning: %s: %s\n", entry.CiteName, warning); err != nil {
			return nil, fmt.Errorf("write BibTeX warning: %w", err)
		}
	}
	return entry, nil
}

// formatBibTeX aligns fields while preserving their contents verbatim.
func formatBibTeX(entry *bibtex.BibEntry) string {
	keys := make([]string, 0, len(entry.Fields))
	width := 0
	for key := range entry.Fields {
		keys = append(keys, key)
		if len(key) > width {
			width = len(key)
		}
	}
	priority := map[string]int{"title": -3, "author": -2, "url": -1}
	sort.Slice(keys, func(i, j int) bool {
		pi, pj := priority[keys[i]], priority[keys[j]]
		return pi < pj || (pi == pj && keys[i] < keys[j])
	})

	var output strings.Builder
	fmt.Fprintf(&output, "@%s{%s,\n", entry.Type, entry.CiteName)
	for _, key := range keys {
		value := entry.Fields[key].String()
		if _, err := strconv.Atoi(value); err != nil {
			// BibTeX delimiters preserve LaTeX commands and newlines; Go's %q escapes them.
			if strings.ContainsAny(value, "\"{}") {
				value = "{" + value + "}"
			} else {
				value = "\"" + value + "\""
			}
		}
		fmt.Fprintf(&output, "    %-*s = %s,\n", width, key, value)
	}
	output.WriteString("}\n")
	return output.String()
}
