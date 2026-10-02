<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/bibi-logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/bibi-logo-light.svg">
    <img alt="bibi" src="assets/bibi-logo-light.svg" width="280">
  </picture>
</div>

---

# bibi

`bibi` is a command-line tool that retrieves BibTeX for mathematical literature
from MathSciNet's limited free MRTools search, [zbMATH Open](https://zbmath.org/),
[Crossref](https://www.crossref.org/), and the
[Mathematics Genealogy Project](https://www.genealogy.math.ndsu.nodak.edu/).

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [Search](#search)
  - [Add to a bibliography](#add-to-a-bibliography)
  - [zbMATH Open](#zbmath-open)
  - [MR Lookup](#mr-lookup)
  - [Mathematics Genealogy Project](#mathematics-genealogy-project)
  - [Bibliographic quality](#bibliographic-quality)
  - [Debugging](#debugging)
- [Development](#development)

## Install

Install the latest tagged version into your Go binary directory:

```sh
go install github.com/thofma/bibi@latest
```

Prebuilt archives for macOS, Linux, and Windows are available from
[GitHub Releases](https://github.com/thofma/bibi/releases).

To build from a checkout, install the Go version declared in [`go.mod`](go.mod),
then run:

```sh
go build -o bibi .
```

## Quick Start

<p align="center">
  <img src="assets/quick-start.gif" alt="Animated terminal showing a bibi zb search and its search spinner" width="560">
</p>

```sh
bibi search "zhang p-adic"
```

If the search returns more than one match, choose a result when prompted. `bibi`
prints its BibTeX entry to standard output.

To save an entry directly into your bibliography:

```sh
bibi add "serre local fields" references.bib --bib mr
```

## Usage

`search`, `zb`, `mr`, and `phd` write a single BibTeX entry to standard output.
`add` saves the entry to the specified file and reports its citation key on
standard error. Search progress,
errors, and the interactive result picker are written to standard error. Press
`q` or `Ctrl-C` to cancel a picker.

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
Crossref discovery uses its
[`query.bibliographic` search](https://api.crossref.org/swagger-ui/index.html).
A year in a free-text query is a search term, not a strict year filter.

The picker shows ten discovery results at a time. Use `n` for the next page and
`p` for a previously loaded page. Further pages are fetched only when requested;
visited pages are cached. A page request failure keeps the current results visible
and offers `n` to retry. Exact DOI lookups have no further pages.

Move with the arrow keys to see complete titles, authors, editors, venue, year,
publication type, DOI, and database identifiers in the detail pane. Source-supplied
edition and version information appears as edition metadata or notes; bibi does
not guess it from a title or an arXiv link. Press `Tab` to scroll the details with
the arrow keys or `PgUp`/`PgDn`, then `Tab` to return to the results. Small terminals
switch between results and details with `Tab`.

The picker uses rounded panels, a highlighted selection, and coloured key hints.
The active panel has a brighter border. Colours adapt to light and dark terminals;
set `NO_COLOR=1` to keep the layout without colours.

Once a work is selected, bibi queries
only the requested BibTeX provider. Candidates with conflicting DOIs are excluded,
and matching DOIs or service identifiers take priority. A single remaining
candidate is returned directly, even when discovery has no DOI. A second picker
opens only when multiple candidates remain. Cancel if the offered records are
different works or editions.

`--bib mr` always returns MR Lookup's BibTeX. `--bib crossref` requires the
selected work to have a DOI and retrieves Crossref's BibTeX export. If the
requested provider cannot supply the entry, the command fails without printing
BibTeX. Crossref is used only when explicitly selected, with no automatic
fallback to another service.

Discovery and export depend on each service's coverage. A work found in one
service may be absent from the chosen BibTeX provider.

MR queries use the first author's family name when discovery supplies a
`Family, Given` name: `Fesenko, I. B.` becomes `Fesenko`, and
`van der Waerden, B. L.` becomes `van der Waerden`. Names without a comma are
kept intact. This changes only the lookup query; the exported BibTeX retains
MR's full author names.

### Add to a bibliography

```sh
bibi add "serre local fields" references.bib --bib mr
bibi add "10.1016/j.jnt.2016.05.016" references.bib --key Hofmann2016
bibi add "local fields" references.bib --discovery crossref --bib mr --dry-run
```

The two positional arguments are the query and the destination file. Quote
multiword queries. `add` uses the same discovery, result picker, BibTeX providers,
and journal preferences as `search`; both discovery and BibTeX default to zbMATH.

The provider's citation key is retained unless you supply `--key`. A new entry
is appended, or a missing file is created if its parent directory exists. A
shared DOI, MR number, zbMATH identifier, Zbl number, or explicitly marked arXiv
identifier identifies an existing entry: bibi reports its existing key and leaves
the file untouched. arXiv versions remain distinct. Matching titles, authors,
or years alone do not establish a duplicate.

Duplicate checks use identifiers from the exported BibTeX. Conflicting identifiers
or a key already used for an unverified work stop the add. Key collisions include
case variants; use `--key` to choose another key. Existing entries are never
updated or merged.

`--dry-run` checks the destination and prints the proposed entry to standard
output without creating or changing any files. If the entry is already present,
it reports the existing key without printing a new entry.

Existing text, comments, string macros, formatting, and LF or CRLF line endings
are preserved. Identifier fields can use defined string macros and concatenation;
unresolved macros in ordinary fields, such as external journal abbreviations,
are kept untouched. Malformed files or identifier values that cannot be safely
evaluated produce an error before saving. Existing symlinks are followed and file
permissions are preserved; new files have private permissions.

The file is checked again after selection. Concurrent bibi adds are coordinated
with a lock in the OS temporary directory. Writes use a flushed temporary file
in the destination directory, and detected outside edits during saving stop the
write. Cancellation or a failed lookup leaves the destination untouched.

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

All lookups require network access to their respective services.

The original `zb`, `mr`, and `phd` commands remain available.

### Bibliographic quality

Generated zbMATH and thesis entries protect title and book-title capitalization
with braces, preserve mathematics and existing TeX commands, retain accents and
contributor names, and escape special characters in text. zbMATH articles include
issue numbers when supplied, both journal names, and TeX page ranges such as
`86--102`. Proceedings articles retain enclosing-book titles and editors when
available in the response. Native MR and Crossref field contents are preserved.

Choose a journal-name preference for any lookup:

```sh
bibi search "10.1016/j.jnt.2016.05.016" --bib mr --journal full
bibi search "10.1016/j.jnt.2016.05.016" --journal short
```

`--journal source` is the default and keeps the provider's chosen journal name.
`full` and `short` use the corresponding name supplied by that provider. Crossref
abbreviations are retained from Crossref discovery when supplied. If a requested
form is unavailable, bibi warns and keeps the original journal name; it never
invents an abbreviation. Other fields and the original citation key are retained.

The final entry is checked for missing
[standard BibTeX required fields](https://mirrors.ctan.org/biblio/bibtex/base/btxdoc.pdf).
For example, a proceedings article without `booktitle` produces:

```text
Warning: zbMATH123: missing required BibTeX fields: booktitle
```

Warnings go to standard error. The entry is still exported or added to the file,
and a successful retrieval still exits successfully, with no extra selection.
Edited books can use an editor instead of an author; DOI, issue number, volume,
and pages are optional. For preprints represented as `misc`, bibi checks that a
title exists without requiring a journal.

### Debugging

Add `--debug` to any lookup command:

```sh
bibi search "local fields" --bib mr --debug
bibi --debug search "10.1016/j.jnt.2016.05.016" --discovery crossref --bib mr
bibi mr serre "a course in arithmetic" 1973 --debug
bibi add "local fields" references.bib --bib mr --debug
```

Use this to diagnose when and why `--bib` fails. The trace shows the selected
work's complete title, authors, year, DOI, and service IDs; the provider's lookup
strategy and query; and returned candidates with reasons for accepting or
rejecting them. It distinguishes an unattempted lookup, an empty response,
conflicting DOIs, malformed exports, confirmation cancellation, and output errors.

For MR, the trace explicitly shows the author/title/year query: the selected
author's name and the family name sent to MR are shown separately. The selected
DOI is used for validation, rather than sent to MR Lookup. For zbMATH, it includes
native records discarded inside the adapter and unsupported document types.
Provider responses and exports appear as quoted excerpts capped at 2048 bytes,
alongside request URLs, HTTP status, and timing. Spinners are disabled during
debug runs.

For `add`, the trace also identifies preflight checks, verified duplicates, and
save failures after a successful provider lookup.

Diagnostics go to standard error, leaving standard output available for BibTeX:

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

`lib/bibliography` defines the provider-neutral `Work`, `Discoverer`, and
`Provider` interfaces. New backends implement those interfaces and are registered
in `cmd/search.go`; the selection and output flow does not depend on their native
response formats. The zbMATH adapter retains its native search records for
BibTeX generation, including related book metadata.
