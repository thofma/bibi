package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

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

var errSelectionCancelled = errors.New("selection cancelled")

func defaultSearchServices() searchServices {
	zbBackend := &zb.Backend{}
	crossrefBackend := &crossref.Backend{}
	return searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": zbBackend, "crossref": crossrefBackend},
		bib:       map[string]bibliography.Provider{"zb": zbBackend, "mr": mr.Provider{}, "crossref": crossrefBackend},
	}
}

func newSearchCommand(services func() searchServices, choose func(util.ChooserRequest) (int, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "search <search terms...>",
		Short: "Find a work and retrieve BibTeX from your chosen provider",
		Long: `Search freely by author, title, year, or other citation terms, or supply a DOI.

Discovery and BibTeX retrieval are independent. Both default to zbMATH Open.
The requested BibTeX provider is always used; bibi never substitutes another.
MR BibTeX comes from the free MR Lookup service.
Unverified provider matches always require confirmation. Verified single matches
are selected automatically unless their edition or publication status changes.

Examples:

  bibi search "serre local fields"
  bibi search "serre local fields" --bib mr
  bibi search "serre local fields" --discovery crossref --bib mr
  bibi search "10.1016/j.jnt.2016.05.016" --bib crossref`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
	}
	addSearchFlags(command)
	command.RunE = withDebug(func(cmd *cobra.Command, args []string) error {
		if _, err := journalStyle(cmd); err != nil {
			return err
		}
		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			return fmt.Errorf("search query cannot be empty")
		}

		record, err := retrieveCitation(cmd, query, services, choose)
		if err != nil {
			return err
		}
		bibName, _ := cmd.Flags().GetString("bib")
		diagnostics.Printf("writing %s BibTeX entry %s", bibName, record.Entry.CiteName)
		if err := writeBibTeX(cmd, record.Entry, record.Journals); err != nil {
			diagnostics.Printf("bib provider=%s stage=output failed: %v; provider lookup and matching succeeded", bibName, err)
			return err
		}
		diagnostics.Printf("bib provider=%s stage=output succeeded: key=%q", bibName, record.Entry.CiteName)
		return nil
	})
	return command
}

func addSearchFlags(command *cobra.Command) {
	command.Flags().String("discovery", "zb", "Discovery backend: zb, crossref")
	command.Flags().String("bib", "zb", "BibTeX provider: zb, mr, crossref")
}

func serviceDisplayName(name string) string {
	switch name {
	case "zb":
		return "zbMATH Open"
	case "mr":
		return "MR Lookup"
	case "crossref":
		return "Crossref"
	default:
		return name
	}
}

// retrieveCitation shares discovery, paging, selection and provider matching
// between commands. The caller decides where the final entry is written.
func retrieveCitation(cmd *cobra.Command, query string, services func() searchServices, choose func(util.ChooserRequest) (int, error)) (bibliography.Record, error) {
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
		return bibliography.Record{}, fmt.Errorf("unknown discovery backend %q (choose zb or crossref)", discoveryName)
	}
	provider, ok := backends.bib[bibName]
	if !ok {
		return bibliography.Record{}, fmt.Errorf("unknown BibTeX provider %q (choose zb, mr, or crossref)", bibName)
	}
	if sized, ok := discoverer.(bibliography.PageSizeSetter); ok {
		pageSize := util.ChooserPageSize(cmd.ErrOrStderr())
		sized.SetPageSize(pageSize)
		diagnostics.Printf("discovery %s page_size=%d", discoveryName, pageSize)
	}

	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Searching "+serviceDisplayName(discoveryName)+"...")
	requestContext := spinner.Context(cmd.Context())
	var page bibliography.SearchPage
	var err error
	paged, canPage := discoverer.(bibliography.PagedDiscoverer)
	if canPage {
		page, err = paged.SearchPage(requestContext, query, "")
	} else {
		page.Works, err = discoverer.Search(query)
	}
	spinner.Stop()
	if err != nil {
		diagnostics.Printf("bib provider=%s not queried: discovery request failed: %v", bibName, err)
		return bibliography.Record{}, fmt.Errorf("search %s: %w", discoveryName, err)
	}
	works := page.Works
	diagnostics.Printf("discovery %s returned %d works", discoveryName, len(works))
	if len(works) == 0 {
		diagnostics.Printf("bib provider=%s not queried: discovery returned no works", bibName)
		return bibliography.Record{}, fmt.Errorf("no %s results found for %q", discoveryName, query)
	}
	var worksMu sync.Mutex
	selected := 0
	if len(works) > 1 || page.NextToken != "" {
		request := util.ChooserRequest{Title: "Choose " + discoveryName + " result: " + query,
			ChoicePage: workChoices(page), Context: cmd.Context(), Output: cmd.ErrOrStderr()}
		if canPage {
			request.LoadPage = func(ctx context.Context, token string) (util.ChoicePage, error) {
				diagnostics.Printf("discovery %s loading next page token=%q", discoveryName, token)
				next, err := paged.SearchPage(ctx, query, token)
				if err != nil {
					diagnostics.Printf("discovery %s next page failed: %v", discoveryName, err)
					return util.ChoicePage{}, err
				}
				if err := ctx.Err(); err != nil {
					return util.ChoicePage{}, err
				}
				worksMu.Lock()
				works = append(works, next.Works...)
				worksMu.Unlock()
				diagnostics.Printf("discovery %s next page returned %d works", discoveryName, len(next.Works))
				return workChoices(next), nil
			}
		}
		selected, err = selectSearchResult(request, choose)
		if err != nil {
			diagnostics.Printf("bib provider=%s not queried: discovery selection failed: %v", bibName, err)
			return bibliography.Record{}, fmt.Errorf("choose discovery result: %w", err)
		}
	}
	worksMu.Lock()
	if selected >= len(works) {
		worksMu.Unlock()
		return bibliography.Record{}, fmt.Errorf("invalid selection %d", selected)
	}
	work := works[selected]
	worksMu.Unlock()
	diagnostics.Printf("selected discovery result %d: %q DOI=%q IDs=%v", selected+1, work.Label(), work.DOI, work.IDs)
	return retrieveProviderCitation(cmd, work, bibName, provider, choose)
}

// retrieveProviderCitation matches and confirms exports from one explicitly
// selected provider, for either a discovered work or an exact identifier lookup.
func retrieveProviderCitation(cmd *cobra.Command, work bibliography.Work, bibName string, provider bibliography.Provider, choose func(util.ChooserRequest) (int, error)) (bibliography.Record, error) {
	diagnostics.Printf("bib provider=%s stage=retrieval started: title=%q authors=%q year=%q DOI=%q normalized_DOI=%q IDs=%v", bibName, work.Title, work.Authors, work.Year, work.DOI, bibliography.NormalizeDOI(work.DOI), work.IDs)
	spinner := util.StartSpinner(cmd.ErrOrStderr(), "Retrieving "+serviceDisplayName(bibName)+" BibTeX...")
	var records []bibliography.Record
	var err error
	if contextual, ok := provider.(bibliography.ContextProvider); ok {
		records, err = contextual.BibTeXContext(spinner.Context(cmd.Context()), work)
	} else {
		records, err = provider.BibTeX(work)
	}
	spinner.Stop()
	if contextErr := cmd.Context().Err(); contextErr != nil {
		return bibliography.Record{}, contextErr
	}
	if err != nil {
		diagnostics.Printf("bib provider=%s stage=retrieval failed: %v", bibName, err)
		return bibliography.Record{}, fmt.Errorf("retrieve %s BibTeX: %w", bibName, err)
	}
	diagnostics.Printf("BibTeX provider %s returned %d candidates", bibName, len(records))
	candidateCount := len(records)
	var compatible, exact []bibliography.Record
	rejected := 0
	for i, record := range records {
		diagnostics.Printf("bib provider=%s candidate=%d title=%q authors=%q year=%q DOI=%q normalized_DOI=%q IDs=%v", bibName, i+1, record.Title, record.Authors, record.Year, record.DOI, bibliography.NormalizeDOI(record.DOI), record.IDs)
		if record.Entry == nil {
			diagnostics.Printf("bib provider=%s stage=export failed: candidate %d contains metadata but no BibTeX entry", bibName, i+1)
			return bibliography.Record{}, fmt.Errorf("%s returned a result without BibTeX", bibName)
		}
		assessment := bibliography.AssessMatch(work, record.Work, bibName)
		switch assessment.Status {
		case bibliography.MatchVerified:
			diagnostics.Printf("candidate %s: exact identifier match: %s", record.Entry.CiteName, assessment.Reason)
			exact = append(exact, record)
		case bibliography.MatchUnverified:
			diagnostics.Printf("candidate %s: compatible without identifier verification: %s", record.Entry.CiteName, assessment.Reason)
			compatible = append(compatible, record)
		case bibliography.MatchConflict:
			rejected++
			diagnostics.Printf("candidate %s: rejected, %s", record.Entry.CiteName, assessment.Reason)
		}
		if assessment.Status != bibliography.MatchConflict {
			for _, reason := range bibliography.ReviewReasons(work, record.Work) {
				diagnostics.Printf("candidate %s: review required: %s", record.Entry.CiteName, reason)
			}
		}
	}
	selected := 0
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
		return bibliography.Record{}, fmt.Errorf("no %s BibTeX match for %q", bibName, work.Title)
	}
	needsConfirmation := len(records) > 1 || len(exact) == 0 || len(bibliography.ReviewReasons(work, records[0].Work)) > 0
	if needsConfirmation {
		diagnostics.Printf("bib provider=%s stage=confirmation started: %d remaining candidates", bibName, len(records))
		diagnostics.Printf("asking to confirm a BibTeX candidate (%d choices)", len(records))
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Confirm %s BibTeX (Enter to use, q to cancel).\n", bibName); err != nil {
			return bibliography.Record{}, err
		}
		choices := make([]util.Choice, len(records))
		for i, record := range records {
			assessment := bibliography.AssessMatch(work, record.Work, bibName)
			choices[i] = util.Choice{Label: "[" + assessment.Label() + "] " + record.Label() + " [" + record.Entry.CiteName + "]",
				Details: bibliography.ComparisonDetails(work, record.Work, assessment) + "\n\nCitation key: " + record.Entry.CiteName}
		}
		selected, err = selectSearchResult(util.ChooserRequest{Title: "Confirm " + bibName + " BibTeX",
			ChoicePage: util.ChoicePage{Choices: choices}, Context: cmd.Context(), Output: cmd.ErrOrStderr(), Confirmation: true}, choose)
		if err != nil {
			diagnostics.Printf("bib provider=%s stage=confirmation failed: %v; provider lookup succeeded", bibName, err)
			return bibliography.Record{}, fmt.Errorf("choose %s BibTeX match: %w", bibName, err)
		}
		if selected >= len(records) {
			return bibliography.Record{}, fmt.Errorf("invalid selection %d", selected)
		}
		diagnostics.Printf("bib provider=%s stage=confirmation succeeded: key=%q", bibName, records[selected].Entry.CiteName)
	} else {
		diagnostics.Printf("bib provider=%s stage=selection automatic: one verified candidate with no edition or publication change", bibName)
	}
	if err := cmd.Context().Err(); err != nil {
		return bibliography.Record{}, err
	}
	return records[selected], nil
}

func workChoices(page bibliography.SearchPage) util.ChoicePage {
	choices := make([]util.Choice, len(page.Works))
	for i, work := range page.Works {
		choices[i] = util.Choice{Label: work.Label(), Details: work.Details()}
	}
	return util.ChoicePage{Choices: choices, NextToken: page.NextToken, Total: page.Total}
}

func selectSearchResult(request util.ChooserRequest, choose func(util.ChooserRequest) (int, error)) (int, error) {
	if request.Context != nil {
		if err := request.Context.Err(); err != nil {
			return 0, err
		}
	}
	if choose == nil {
		choose = util.RunDetailedChooser
	}
	selected, err := choose(request)
	if err != nil {
		return 0, err
	}
	if request.Context != nil {
		if err := request.Context.Err(); err != nil {
			return 0, err
		}
	}
	if selected < 0 {
		return 0, errSelectionCancelled
	}
	return selected, nil
}

func init() {
	rootCmd.AddCommand(newSearchCommand(defaultSearchServices, util.RunDetailedChooser))
}
