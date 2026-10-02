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

## Usage

Each command writes a single BibTeX entry to standard output. Search progress,
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

Choose among up to ten discovery results. Once a work is selected, bibi queries
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

Warnings go to standard error. The entry is still printed to standard output,
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
