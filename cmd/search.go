package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/crossref"
	"github.com/thofma/bibi/lib/mr"
	"github.com/thofma/bibi/lib/zb"
	"github.com/thofma/bibi/util"
)

// searchServices keeps discovery independent of the requested BibTeX provider.
type searchServices struct {
	discovery map[string]bibliography.Discoverer
	bib       map[string]bibliography.Provider
}

func defaultSearchServices() searchServices {
	zbBackend := &zb.Backend{}
	crossrefBackend := &crossref.Backend{}
	return searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": zbBackend, "crossref": crossrefBackend},
		bib:       map[string]bibliography.Provider{"zb": zbBackend, "mr": mr.Provider{}, "crossref": crossrefBackend},
	}
}

func newSearchCommand(services func() searchServices, choose func([]string) (int, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "search <search terms...>",
		Short: "Find a work and retrieve BibTeX from your chosen provider",
		Long: `Search freely by author, title, year, or other citation terms, or supply a DOI.

Discovery and BibTeX retrieval are independent. Both default to zbMATH Open.
The requested BibTeX provider is always used; bibi never substitutes another.
MR BibTeX comes from the free MR Lookup service.

Examples:

  bibi search "serre local fields"
  bibi search "serre local fields" --bib mr
  bibi search "serre local fields" --discovery crossref --bib mr
  bibi search "10.1016/j.jnt.2016.05.016" --bib crossref`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
	}
	command.Flags().String("discovery", "zb", "Discovery backend: zb, crossref")
	command.Flags().String("bib", "zb", "BibTeX provider: zb, mr, crossref")
	command.RunE = withDebug(func(cmd *cobra.Command, args []string) error {
		if _, err := journalStyle(cmd); err != nil {
			return err
		}
		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			return fmt.Errorf("search query cannot be empty")
		}
		discoveryName, _ := cmd.Flags().GetString("discovery")
		bibName, _ := cmd.Flags().GetString("bib")
		queryType := "free text"
		if _, isDOI := bibliography.DOIQuery(query); isDOI {
			queryType = "DOI"
		}
		diagnostics.Printf("discovery=%s bib=%s query=%q query_type=%q", discoveryName, bibName, query, queryType)
		backends := services()
		discoverer, ok := backends.discovery[discoveryName]
		if !ok {
			return fmt.Errorf("unknown discovery backend %q (choose zb or crossref)", discoveryName)
		}
		provider, ok := backends.bib[bibName]
		if !ok {
			return fmt.Errorf("unknown BibTeX provider %q (choose zb, mr, or crossref)", bibName)
		}

		spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching "+discoveryName+"...")
		works, err := discoverer.Search(query)
		spinner.Stop()
		if err != nil {
			diagnostics.Printf("bib provider=%s not queried: discovery request failed: %v", bibName, err)
			return fmt.Errorf("search %s: %w", discoveryName, err)
		}
		diagnostics.Printf("discovery %s returned %d works", discoveryName, len(works))
		if len(works) == 0 {
			diagnostics.Printf("bib provider=%s not queried: discovery returned no works", bibName)
			return fmt.Errorf("no %s results found for %q", discoveryName, query)
		}
		if len(works) > bibliography.MaxResults {
			works = works[:bibliography.MaxResults]
		}
		selected := 0
		if len(works) > 1 {
			labels := make([]string, len(works))
			for i, work := range works {
				labels[i] = work.Label()
			}
			selected, err = selectSearchResult(labels, choose)
			if err != nil {
				diagnostics.Printf("bib provider=%s not queried: discovery selection failed: %v", bibName, err)
				return fmt.Errorf("choose discovery result: %w", err)
			}
		}
		work := works[selected]
		diagnostics.Printf("selected discovery result %d: %q DOI=%q IDs=%v", selected+1, work.Label(), work.DOI, work.IDs)
		diagnostics.Printf("bib provider=%s stage=retrieval started: title=%q authors=%q year=%q DOI=%q normalized_DOI=%q IDs=%v", bibName, work.Title, work.Authors, work.Year, work.DOI, bibliography.NormalizeDOI(work.DOI), work.IDs)
		spinner = util.StartSpinner(cmd.ErrOrStderr(), "Retrieving "+bibName+" BibTeX...")
		records, err := provider.BibTeX(work)
		spinner.Stop()
		if err != nil {
			diagnostics.Printf("bib provider=%s stage=retrieval failed: %v", bibName, err)
			return fmt.Errorf("retrieve %s BibTeX: %w", bibName, err)
		}
		diagnostics.Printf("BibTeX provider %s returned %d candidates", bibName, len(records))
		candidateCount := len(records)
		var compatible, exact []bibliography.Record
		rejected := 0
		for i, record := range records {
			diagnostics.Printf("bib provider=%s candidate=%d title=%q authors=%q year=%q DOI=%q normalized_DOI=%q IDs=%v", bibName, i+1, record.Title, record.Authors, record.Year, record.DOI, bibliography.NormalizeDOI(record.DOI), record.IDs)
			if record.Entry == nil {
				diagnostics.Printf("bib provider=%s stage=export failed: candidate %d contains metadata but no BibTeX entry", bibName, i+1)
				return fmt.Errorf("%s returned a result without BibTeX", bibName)
			}
			isExact, isCompatible := bibliography.Match(work, record.Work, bibName)
			if isExact {
				diagnostics.Printf("candidate %s: exact identifier match: %s", record.Entry.CiteName, bibliography.MatchReason(work, record.Work, bibName))
				exact = append(exact, record)
			} else if isCompatible {
				diagnostics.Printf("candidate %s: compatible without identifier verification: %s", record.Entry.CiteName, bibliography.MatchReason(work, record.Work, bibName))
				compatible = append(compatible, record)
			} else {
				rejected++
				diagnostics.Printf("candidate %s: rejected, %s", record.Entry.CiteName, bibliography.MatchReason(work, record.Work, bibName))
			}
		}
		selected = 0
		diagnostics.Printf("bib provider=%s stage=matching exact=%d compatible=%d rejected=%d", bibName, len(exact), len(compatible), rejected)
		if len(exact) > 0 {
			records = exact
		} else {
			records = compatible
		}
		if len(records) == 0 {
			if candidateCount == 0 {
				diagnostics.Printf("bib provider=%s stage=retrieval failed: provider supplied no usable BibTeX candidates; see provider lookup, parsing, and matching trace above", bibName)
			} else {
				diagnostics.Printf("bib provider=%s stage=matching failed: all %d candidates were rejected for conflicting DOIs", bibName, candidateCount)
			}
			return fmt.Errorf("no %s BibTeX match for %q", bibName, work.Title)
		}
		if len(records) > bibliography.MaxResults {
			records = records[:bibliography.MaxResults]
		}
		if len(records) > 1 {
			diagnostics.Printf("bib provider=%s stage=confirmation started: %d remaining candidates", bibName, len(records))
			diagnostics.Printf("asking to confirm a BibTeX candidate (%d choices)", len(records))
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Confirm %s BibTeX for: %s\n", bibName, work.Label()); err != nil {
				return err
			}
			labels := make([]string, len(records))
			for i, record := range records {
				labels[i] = record.Label() + " [" + record.Entry.CiteName + "]"
				if record.DOI != "" {
					labels[i] += " DOI: " + record.DOI
				}
			}
			selected, err = selectSearchResult(labels, choose)
			if err != nil {
				diagnostics.Printf("bib provider=%s stage=confirmation failed: %v; provider lookup succeeded", bibName, err)
				return fmt.Errorf("choose %s BibTeX match: %w", bibName, err)
			}
		} else {
			diagnostics.Printf("bib provider=%s stage=selection automatic: one remaining candidate", bibName)
		}
		diagnostics.Printf("writing %s BibTeX entry %s", bibName, records[selected].Entry.CiteName)
		if err := writeBibTeX(cmd, records[selected].Entry, records[selected].Journals); err != nil {
			diagnostics.Printf("bib provider=%s stage=output failed: %v; provider lookup and matching succeeded", bibName, err)
			return err
		}
		diagnostics.Printf("bib provider=%s stage=output succeeded: key=%q", bibName, records[selected].Entry.CiteName)
		return nil
	})
	return command
}

func selectSearchResult(labels []string, choose func([]string) (int, error)) (int, error) {
	selected, err := choose(labels)
	if err != nil {
		return 0, err
	}
	if selected < 0 {
		return 0, fmt.Errorf("selection cancelled")
	}
	if selected >= len(labels) {
		return 0, fmt.Errorf("invalid selection %d", selected)
	}
	return selected, nil
}

func init() {
	rootCmd.AddCommand(newSearchCommand(defaultSearchServices, util.RunChooser))
}
