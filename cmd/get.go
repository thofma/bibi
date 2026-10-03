package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/arxiv"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/doi"
	"github.com/thofma/bibi/util"
)

type getServices struct {
	arxiv       func(context.Context, string) (bibliography.Record, error)
	doi         func(context.Context, string) (bibliography.Record, error)
	doiMetadata func(context.Context, string) (bibliography.Work, error)
	bib         map[string]bibliography.Provider
}

func defaultGetServices() getServices {
	arxivBackend := &arxiv.Backend{}
	doiBackend := &doi.Backend{}
	return getServices{arxiv: paceArXivLookups(arxivBackend.Lookup, 3*time.Second), doi: doiBackend.Lookup,
		doiMetadata: doiBackend.Metadata, bib: defaultSearchServices().bib}
}

// Batch lookups run sequentially. Space arXiv requests according to its API
// guidance, allowing cancellation while waiting for the next request.
func paceArXivLookups(lookup func(context.Context, string) (bibliography.Record, error), interval time.Duration) func(context.Context, string) (bibliography.Record, error) {
	var next time.Time
	return func(ctx context.Context, id string) (bibliography.Record, error) {
		if err := ctx.Err(); err != nil {
			return bibliography.Record{}, err
		}
		if delay := time.Until(next); delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return bibliography.Record{}, ctx.Err()
			case <-timer.C:
			}
		}
		if err := ctx.Err(); err != nil {
			return bibliography.Record{}, err
		}
		record, err := lookup(ctx, id)
		next = time.Now().Add(interval)
		return record, err
	}
}

func newGetCommand(services func() getServices, choose func(util.ChooserRequest) (int, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "get [identifier-or-url ...]",
		Short: "Retrieve BibTeX directly for DOI or arXiv identifiers or URLs",
		Long: `Retrieve exact references without searching a discovery database.

Supply one or more identifiers as arguments, or omit arguments to read one
identifier per line from piped or redirected stdin. Blank lines and repeated
normalized identifiers are skipped; distinct arXiv versions remain separate.
Arguments take precedence over stdin. Without arguments or redirected input,
get shows usage. Successful entries are printed in input order; individual
failures are reported on stderr and result in a nonzero exit status.

DOIs use doi.org content negotiation, including Crossref and DataCite records.
arXiv identifiers and abstract, PDF or HTML URLs return a generated preprint
entry. Explicit arXiv versions are preserved; unversioned IDs use the latest.

Use --published with an arXiv input to retrieve the article identified by its
supplied publication DOI. An explicit --bib provider supplies its own BibTeX,
with no fallback. Use search for free-text author/title queries.

Examples:

  bibi get 2301.12345
  bibi get arXiv:2301.12345v2
  bibi get https://arxiv.org/pdf/2301.12345v2
  bibi get math/0303109
  bibi get https://doi.org/10.1016/j.jnt.2016.05.016
  bibi get math/0211159 10.1016/j.jnt.2016.05.016
  bibi get < identifiers.txt
  pbpaste | bibi get > references.bib
  bibi get 1701.00340 --published
  bibi get 10.1016/j.jnt.2016.05.016 --bib mr`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}
	command.Flags().String("bib", "auto", "BibTeX provider: auto (arXiv or DOI service), zb, mr, crossref")
	command.Flags().Bool("published", false, "Follow the publication DOI supplied by arXiv")
	command.RunE = withDebug(func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, args, services, choose)
	})
	return command
}

func retrieveGetCitation(cmd *cobra.Command, identifier bibliography.Identifier, bibName string, published bool, backends getServices, choose func(util.ChooserRequest) (int, error)) (bibliography.Record, error) {
	diagnostics.Printf("get identifier_type=%s identifier=%q bib=%s published=%t", identifier.Kind, identifier.Value, bibName, published)
	var record bibliography.Record
	var err error
	if identifier.Kind == "arxiv" {
		spinner := util.StartSpinner(cmd.ErrOrStderr(), "Retrieving arXiv metadata...")
		record, err = backends.arxiv(cmd.Context(), identifier.Value)
		spinner.Stop()
		if err != nil {
			return bibliography.Record{}, err
		}
		if published {
			publicationDOI, valid := bibliography.DOIQuery(record.DOI)
			if !valid {
				return bibliography.Record{}, fmt.Errorf("arXiv %s supplies no publication DOI; retrieve the preprint without --published", identifier.Value)
			}
			identifier = bibliography.Identifier{Kind: "doi", Value: publicationDOI}
			diagnostics.Printf("get following arXiv publication DOI=%q", publicationDOI)
		}
	}
	if err := cmd.Context().Err(); err != nil {
		return bibliography.Record{}, err
	}
	if identifier.Kind == "doi" {
		if bibName == "auto" {
			spinner := util.StartSpinner(cmd.ErrOrStderr(), "Retrieving DOI BibTeX...")
			record, err = backends.doi(cmd.Context(), identifier.Value)
			spinner.Stop()
		} else {
			spinner := util.StartSpinner(cmd.ErrOrStderr(), "Retrieving DOI metadata...")
			record.Work, err = backends.doiMetadata(cmd.Context(), identifier.Value)
			spinner.Stop()
		}
		if err != nil {
			return bibliography.Record{}, err
		}
	}
	if err := cmd.Context().Err(); err != nil {
		return bibliography.Record{}, err
	}
	diagnostics.Printf("get resolved title=%q authors=%q year=%q DOI=%q IDs=%v", record.Title, record.Authors, record.Year, record.DOI, record.IDs)
	if bibName != "auto" {
		record, err = retrieveProviderCitation(cmd, record.Work, bibName, backends.bib[bibName], choose)
		if err != nil {
			return bibliography.Record{}, err
		}
	}
	if record.Entry == nil {
		return bibliography.Record{}, fmt.Errorf("lookup returned metadata without a BibTeX entry")
	}
	return record, nil
}

func init() {
	rootCmd.AddCommand(newGetCommand(defaultGetServices, util.RunDetailedChooser))
}
