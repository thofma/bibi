package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/util"
)

func getInputIsTerminal(input io.Reader) bool {
	file, ok := input.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}

// Preserve underlying lookup errors for callers while keeping Cobra's final
// error concise. Individual failures have already been reported on stderr.
type getBatchError struct {
	failures []error
	written  int
}

func (err *getBatchError) Error() string {
	input, entry := "inputs", "entries"
	if len(err.failures) == 1 {
		input = "input"
	}
	if err.written == 1 {
		entry = "entry"
	}
	return fmt.Sprintf("%d %s failed; %d BibTeX %s written", len(err.failures), input, err.written, entry)
}

func (err *getBatchError) Unwrap() []error { return err.failures }

func runGet(cmd *cobra.Command, args []string, services func() getServices, choose func(util.ChooserRequest) (int, error)) error {
	if _, err := journalStyle(cmd); err != nil {
		return err
	}
	bibName, _ := cmd.Flags().GetString("bib")
	switch bibName {
	case "auto", "zb", "mr", "crossref":
	default:
		return fmt.Errorf("unknown BibTeX provider %q (choose auto, zb, mr, or crossref)", bibName)
	}
	published, _ := cmd.Flags().GetBool("published")
	input := cmd.InOrStdin()
	if len(args) == 0 && getInputIsTerminal(input) {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), cmd.UsageString()); err != nil {
			return fmt.Errorf("write get usage: %w", err)
		}
		return fmt.Errorf("provide a DOI or arXiv identifier, or pipe one identifier per line into bibi get")
	}

	seen := make(map[bibliography.Identifier]bool)
	keys := make(map[string]bibliography.Identifier)
	batch := &getBatchError{}
	var backends getServices
	initialized := false
	reportFailure := func(label string, err error) error {
		if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s: %v\n", label, err); writeErr != nil {
			return fmt.Errorf("report get failure: %w", writeErr)
		}
		batch.failures = append(batch.failures, fmt.Errorf("%s: %w", label, err))
		return nil
	}
	consume := func(value, label string) error {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		identifier, err := bibliography.ParseIdentifier(value)
		if err != nil {
			return reportFailure(label, err)
		}
		if seen[identifier] {
			diagnostics.Printf("get skipping repeated identifier_type=%s identifier=%q", identifier.Kind, identifier.Value)
			return nil
		}
		seen[identifier] = true
		if published && identifier.Kind != "arxiv" {
			return reportFailure(label, fmt.Errorf("--published requires an arXiv identifier or URL"))
		}
		if !initialized {
			backends = services()
			if bibName != "auto" && backends.bib[bibName] == nil {
				return fmt.Errorf("BibTeX provider %q is unavailable", bibName)
			}
			initialized = true
		}
		record, err := retrieveGetCitation(cmd, identifier, bibName, published, backends, choose)
		if contextErr := cmd.Context().Err(); contextErr != nil {
			return contextErr
		}
		if err != nil {
			if errors.Is(err, errSelectionCancelled) {
				return err
			}
			return reportFailure(label, err)
		}
		key := strings.ToLower(record.Entry.CiteName)
		if previous, exists := keys[key]; exists {
			return reportFailure(label, fmt.Errorf("citation key %q is already used by %s %s", record.Entry.CiteName, previous.Kind, previous.Value))
		}
		diagnostics.Printf("get writing BibTeX key=%q", record.Entry.CiteName)
		if err := writeBibTeX(cmd, record.Entry, record.Journals); err != nil {
			return err // Broken output or diagnostic streams stop the batch.
		}
		keys[key] = identifier
		batch.written++
		return nil
	}

	if len(args) > 0 {
		for _, value := range args {
			if err := consume(value, fmt.Sprintf("%q", value)); err != nil {
				return err
			}
		}
	} else {
		scanner := bufio.NewScanner(input)
		line := 0
		for scanner.Scan() {
			line++
			value := strings.TrimSpace(scanner.Text())
			if value == "" {
				continue
			}
			if err := consume(value, fmt.Sprintf("stdin line %d (%q)", line, value)); err != nil {
				return err
			}
		}
		if err := scanner.Err(); err != nil {
			readErr := fmt.Errorf("read identifiers from stdin: %w", err)
			if len(batch.failures) > 0 {
				return errors.Join(readErr, batch)
			}
			return readErr
		}
		if len(seen) == 0 && len(batch.failures) == 0 {
			return fmt.Errorf("stdin contained no DOI or arXiv identifiers")
		}
	}
	if len(batch.failures) > 0 {
		return batch
	}
	return nil
}
