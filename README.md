<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/bibi-logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/bibi-logo-light.svg">
    <img alt="bibi" src="assets/bibi-logo-light.svg" width="280">
  </picture>
</div>

---

# bibi

`bibi` is a command-line tool for:

- Retrieving BibTeX directly for DOIs and [arXiv](https://arxiv.org/) preprints.
- Searching mathematical literature and retrieving BibTeX from
  [zbMATH Open](https://zbmath.org/), the
  [Mathematics Genealogy Project](https://www.genealogy.math.ndsu.nodak.edu/),
  MathSciNet (through MR Lookup), or [Crossref](https://www.crossref.org/).
- Finding journal abbreviations offline.

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [Get by identifier or URL](#get-by-identifier-or-url)
  - [Search](#search)
  - [Add to a bibliography](#add-to-a-bibliography)
  - [Journal abbreviations](#journal-abbreviations)
  - [zbMATH Open](#zbmath-open)
  - [MR Lookup](#mr-lookup)
  - [Mathematics Genealogy Project](#mathematics-genealogy-project)
  - [BibTeX options](#bibtex-options)
  - [Network recovery](#network-recovery)
  - [Debugging](#debugging)
- [Development](#development)

## Install

Install with Homebrew on macOS or Linux:

```sh
brew install thofma/tap/bibi
```

Homebrew builds `bibi` from source, manages the Go build dependency, and installs
shell completions. The tap is updated automatically after each stable release.

Alternatively, install the latest tagged version with Go:

```sh
go install github.com/thofma/bibi@latest
```

Ensure the install directory (`GOBIN`, or `$(go env GOPATH)/bin` by default)
is on your `PATH` so the `bibi` command is available.

Prebuilt archives for macOS, Linux, and Windows are available from
[GitHub Releases](https://github.com/thofma/bibi/releases).

The examples below describe the current checkout; tagged releases may have fewer
commands. To build from a checkout, use the Go version declared in
[`go.mod`](go.mod) or newer, then run:

```sh
go build -o bibi .
```

Use `./bibi` in the examples below to run this local build, or run `go install .`
to install the checkout into your Go binary directory.

## Quick Start

<p align="center">
  <img src="assets/quick-start.gif" alt="Animated terminal showing a bibi zb search and its search spinner" width="560">
</p>

```sh
bibi search "zhang p-adic"
```

If the search returns more than one match, choose a result when prompted. `bibi`
prints its BibTeX entry to standard output.

If you already have a DOI or arXiv link, retrieve it directly:

```sh
bibi get https://doi.org/10.1016/j.jnt.2016.05.016
bibi get https://arxiv.org/pdf/math/0303109
```

To save an entry directly into your bibliography:

```sh
bibi add "serre local fields" references.bib --bib mr
```

## Usage

Use `bibi --help` to list commands and `bibi <command> --help` for command
options and examples.

`get` writes one BibTeX entry per distinct input to standard output.
`search`, `zb`, `mr`, and `phd` write a single BibTeX entry to standard output.
`add` saves the entry to the specified file and reports its citation key on
standard error. `abbr` prints a journal abbreviation to standard output.
Search progress, errors, and the interactive result picker are written to
standard error. Press `q` or `Ctrl-C` to cancel a picker.

### Get by identifier or URL

```sh
bibi get 2301.12345
bibi get arXiv:2301.12345v2
bibi get https://arxiv.org/pdf/2301.12345v2.pdf
bibi get https://arxiv.org/html/2301.12345v2
bibi get math/0303109
bibi get 10.1016/j.jnt.2016.05.016
bibi get doi:10.1016/j.jnt.2016.05.016
bibi get https://doi.org/10.1016/j.jnt.2016.05.016
```

`get` accepts DOI or arXiv identifiers or URLs and resolves them directly,
without requiring a record in zbMATH or searching by title. Use `search` for
free-text queries. Query strings and fragments on recognized URLs are ignored.

Retrieve several independent references by supplying multiple arguments:

```sh
bibi get math/0211159 10.1016/j.jnt.2016.05.016
```

Without arguments, `get` reads one identifier or supported URL per line from
piped or redirected standard input. No extra flag is needed:

```sh
bibi get < identifiers.txt > references.bib
printf '%s\n' 'math/0211159' '10.1016/j.jnt.2016.05.016' | bibi get
# On macOS, copy a list of identifiers or links first:
pbpaste | bibi get > references.bib
```

To retrieve the published article associated with an arXiv preprint:

```sh
bibi get 1701.00340 --published
bibi get 1701.00340 --published --bib mr
```

`--published` requires arXiv inputs and retrieves the published article using the
publication DOI listed by arXiv. A preprint without a publication DOI produces an
error.

`--bib auto` is the default: arXiv inputs return a preprint entry, and DOI
inputs (including `--published`) return the DOI service's export. You can instead
request a specific provider:

```sh
bibi get 10.1016/j.jnt.2016.05.016 --bib mr
bibi get 10.1016/j.jnt.2016.05.016 --bib crossref
bibi get 2301.12345 --bib zb
```

Explicit choices are `zb`, `mr`, and `crossref`. The selected provider supplies the
BibTeX, with no automatic fallback. Identifier-verified matches are selected
automatically when there is one candidate and no edition or publication change.
Unverified matches always require confirmation, even when there is one candidate.
For arXiv inputs, a provider may return the published work; `--bib auto` returns
the preprint unless `--published` is set. A change from a selected preprint to a
published provider record requires confirmation; `--published` explicitly requests
the published work.

`get` also supports the shared `--journal` and `--debug` options.

### Search

```sh
bibi search "serre local fields 1979"
bibi search "serre local fields" --bib mr
bibi search "serre local fields" --discovery crossref --bib mr
bibi search "10.1016/j.jnt.2016.05.016" --bib crossref
```

Discovery finds the work; `--bib` chooses the service supplying its BibTeX.
The two choices are independent:

| Option | Choices | Default |
| --- | --- | --- |
| `--discovery` | `zb`, `crossref` | `zb` |
| `--bib` | `zb`, `mr`, `crossref` | `zb` |

Queries are free text, so authors, titles, and years can be combined. A bare DOI
or a DOI resolver URL uses an exact DOI lookup in the chosen discovery service.
A year in a free-text query is a search term, not a strict year filter.

The selector fills the terminal height, showing as many results as fit. Use the
arrow keys to highlight a result and `PgUp`/`PgDn` to move through the current
results. Use `n` for the next provider page and `p` for the previous page.
Search requests enough results to fill the initial terminal height, up to 100 per
provider page. That page size stays fixed while browsing so resizing does not skip
results; the selector itself resizes immediately.
Temporary network failures retry automatically while keeping the current results
visible. If loading still fails, press `n` to retry the same page.

Press `Tab` to scroll the detail pane with the arrow keys or `PgUp`/`PgDn`, then
`Tab` to return to the results. Wide terminals show results and details side by
side; narrower or short terminals show one pane at a time, with `Tab` switching
between them. Leaving the selector restores the previous terminal screen.
Set `NO_COLOR=1` to disable colours.

After selecting a work, bibi checks the provider candidates against its identifiers.
A shared normalized DOI or provider identifier verifies identity; conflicting
DOIs reject a candidate. When verified candidates exist, they take precedence over
unverified candidates. Similar titles, authors, or years do not verify identity.

A single verified candidate is selected automatically unless its supplied edition
differs or it changes between preprint and published work. Multiple candidates and
all unverified matches require confirmation, including a single unverified match.
The picker labels the match status and shows its reason, metadata differences, and
the selected work and provider candidate in separate detail sections. Unavailable
fields are marked explicitly. Edition and translation notes are shown when supplied
by the source. Year or title differences are informational and do not by themselves
reject a candidate. Press `Enter` to use the highlighted entry, or `q` or `Ctrl-C`
to cancel without exporting or saving it.

Choices and confirmations read keyboard input from the controlling terminal,
leaving piped identifiers untouched. If no interactive terminal is available, a
required confirmation fails with an actionable error. Batch `get` reports that
input's failure and continues with the remaining inputs; cancelling a picker stops
the batch. Verified single matches still work without a terminal.

The requested BibTeX provider is always used, with no automatic fallback.
`--bib mr` returns MR Lookup's BibTeX, and `--bib crossref` requires a DOI.
If the provider cannot supply the entry, the command fails without printing
BibTeX. Coverage varies by service.

Some zbMATH records replace citation fields with a license restriction notice.
The selector marks these records as restricted and keeps available identifiers
and journal metadata. bibi refuses to export the restricted citation as BibTeX.
For a record with a DOI, the details and error include a command to retrieve it
through Crossref, for example:

```sh
bibi search 10.2307/2306804 --discovery crossref --bib crossref
```

### Add to a bibliography

```sh
bibi add "serre local fields" references.bib --bib mr
bibi add "10.1016/j.jnt.2016.05.016" references.bib --key Hofmann2016
bibi add "local fields" references.bib --discovery crossref --bib mr --dry-run
```

The two positional arguments are the query and the destination file. Quote
multiword queries. `add` uses the same discovery, result picker, BibTeX providers,
and journal preferences as `search`; both discovery and BibTeX default to zbMATH.
The same verification and confirmation rules apply before an entry is saved,
including with `--dry-run`.

The provider's citation key is used unless you supply `--key`. Entries are
appended, or a missing file is created if its parent directory exists. Entries
with matching DOIs or database identifiers are skipped, and their existing keys
are reported. Different arXiv versions remain separate. Conflicting identifiers
or citation keys produce an error. For a citation-key collision, use `--key` to
choose another key.

Existing entries and formatting are preserved.

`--dry-run` checks the destination and prints the proposed entry to standard
output without creating or changing any files. If the entry is already present,
it reports the existing key without printing a new entry.

Malformed bibliography files cause an error. Cancelling or a failed lookup leaves
the destination unchanged.

### Journal abbreviations

```sh
bibi abbr inventiones mathematicae
# Invent. Math.

bibi abbr theory number journal
bibi abbr "reine angewandte"
bibi abbr "ann. math."
```

`abbr` searches a bundled journal abbreviation catalog and works offline. The
catalog is stored as compressed JSON and embedded into the executable; no separate
data file or first-run download is needed. See the
[bundled data notes](internal/journals/README.md) for the format and refresh steps.

Words can appear in any order, either as separate arguments or one quoted query.
Every word must match the beginning of a word in the full title, translated title,
or abbreviation.
For example, `theor numbe jour` also finds Journal of Number Theory. Matching
ignores case, accents, and punctuation.

A unique match prints immediately. Multiple matches open a picker showing
titles, abbreviations, and ISSNs. Only the chosen abbreviation goes to standard
output. Cancelling or finding no matches returns a nonzero status.

### zbMATH Open

```sh
bibi zb "zhang p-adic"
```

`zb` searches [zbMATH Open](https://zbmath.org/) and lets you choose among up to
ten results.

### MR Lookup

```sh
bibi mr serre "a course in arithmetic" 1973
```

`mr` uses MathSciNet's limited free MRTools search, rather than the full
MathSciNet database. The title and year are optional. Use `-` to leave an
earlier field empty, for example `bibi mr serre - 1973`.

### Mathematics Genealogy Project

```sh
bibi phd "carl gauss"
```

`phd` searches the [Mathematics Genealogy Project](https://www.genealogy.math.ndsu.nodak.edu/)
for a mathematician's thesis.

BibTeX lookups require network access to their respective services.

The original `zb`, `mr`, and `phd` commands remain available.

### BibTeX options

Choose a journal-name preference for any lookup:

```sh
bibi search "10.1016/j.jnt.2016.05.016" --bib mr --journal full
bibi search "10.1016/j.jnt.2016.05.016" --journal short
```

`--journal source` is the default and uses the provider's journal name. Choose
`full` or `short` for the corresponding form, when available. If the chosen form
is unavailable, bibi warns and keeps the original journal name.

Incomplete entries produce warnings on standard error. The entry is still
exported or added to the bibliography, and warnings do not change a successful
command's exit status.

### Network recovery

All lookup commands retry temporary connection failures, timeouts, interrupted
responses, and HTTP 408, 429, 500, 502, 503, and 504 responses automatically.
Each request makes at most three attempts, with waits of one and two seconds
plus up to 250 milliseconds of jitter. An attempt has a 15-second timeout, and
the entire request, including waits, has a 45-second limit.

Server `Retry-After` instructions can extend the wait. If that wait exceeds the
request limit, bibi stops and reports when to try again. Later inputs in the same
batch respect the server's cooldown. arXiv requests, including retries, run one
at a time with at least three seconds between their starts.

Retry progress appears in the spinner or result picker, or as plain text on
standard error when output is redirected. Press `Ctrl-C` to cancel a request or
wait and stop the batch. Otherwise, a batch continues after a failed input and
returns a nonzero exit status if any input failed. Invalid requests, permanent
HTTP errors, malformed data, missing records, and license restrictions fail
without retries. An entry is exported or saved only after a successful lookup.

### Debugging

Add `--debug` to any lookup command:

```sh
bibi search "local fields" --bib mr --debug
bibi --debug search "10.1016/j.jnt.2016.05.016" --discovery crossref --bib mr
bibi get https://arxiv.org/pdf/math/0303109 --debug
bibi mr serre "a course in arithmetic" 1973 --debug
bibi add "local fields" references.bib --bib mr --debug
```

Use this to diagnose lookup failures. Queries, results, and errors go to standard
error, leaving standard output available for BibTeX:

```sh
bibi search "10.1016/j.jnt.2016.05.016" --bib mr --debug > entry.bib 2> debug.log
```

## Development

Run the repository checks with:

```sh
go test ./...
go vet ./...
go build ./...
gofmt -l .
```

With BibTeX and pdfLaTeX installed, an optional rendering check verifies the
generated output with the standard `plain` bibliography style:

```sh
BIBI_TEST_TEX=1 go test ./cmd -run TestBibTeXRendering
```
