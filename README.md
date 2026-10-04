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
  - [Other lookup commands](#other-lookup-commands)
  - [BibTeX options](#bibtex-options)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Install

Install with Homebrew on macOS or Linux:

```sh
brew install thofma/tap/bibi
```

Prebuilt archives for macOS, Linux, and Windows are available from
[GitHub Releases](https://github.com/thofma/bibi/releases).

Alternatively, install the latest tagged version with Go:

```sh
go install github.com/thofma/bibi@latest
```

Make sure your Go binary directory (`$(go env GOPATH)/bin` by default) is on
your `PATH` so the `bibi` command is available.

## Quick Start

<p align="center">
  <img src="assets/quick-start.gif" alt="Terminal demo of bibi search for serre local fields, selecting a result, and retrieving BibTeX" width="760">
</p>

```sh
bibi search "serre local fields"
```

Choose a result when prompted. `bibi` prints its BibTeX entry in the terminal.
To save it to a file, run `bibi search "serre local fields" > entry.bib`.

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

### Get by identifier or URL

```sh
bibi get 2301.12345
bibi get arXiv:2301.12345v2
bibi get math/0303109
bibi get 10.1016/j.jnt.2016.05.016
```

`get` accepts DOI and arXiv identifiers or links. Use `search` when you have
an author or title instead.

Retrieve several references at once, or read one identifier or link per line
from a file:

```sh
bibi get math/0211159 10.1016/j.jnt.2016.05.016
bibi get < identifiers.txt > references.bib
# On macOS, copy a list of identifiers or links first:
pbpaste | bibi get > references.bib
```

arXiv inputs return a preprint entry by default. To retrieve the published
article, use `--published` (requires a publication DOI listed by arXiv):

```sh
bibi get 1701.00340 --published
```

To request BibTeX from a particular service, use `--bib zb`, `--bib mr`, or
`--bib crossref`:

```sh
bibi get 10.1016/j.jnt.2016.05.016 --bib mr
```

When using another service, you may be asked to confirm a match or a change
from preprint to published article.

### Search

```sh
bibi search "serre local fields 1979"
bibi search "serre local fields" --bib mr
bibi search "serre local fields" --discovery crossref --bib mr
```

Combine authors, titles, and years in your query, or search by DOI.
By default, `search` uses zbMATH Open. You can choose where to search and
where to retrieve BibTeX separately:

| Option | Purpose | Choices | Default |
| --- | --- | --- | --- |
| `--discovery` | Search service | `zb`, `crossref` | `zb` |
| `--bib` | BibTeX service | `zb`, `mr`, `crossref` | `zb` |

You may be asked to confirm the entry if the match is uncertain or its edition
differs. Review the displayed details before accepting.

Coverage varies by service; bibi reports an error if your chosen BibTeX service
cannot supply the entry. Crossref BibTeX requires a DOI.

### Add to a bibliography

```sh
bibi add "serre local fields" references.bib --bib mr
bibi add "10.1016/j.jnt.2016.05.016" references.bib --key Hofmann2016
bibi add "local fields" references.bib --dry-run
```

`add` takes a query and a destination file. Quote multiword queries. It uses
the same search and BibTeX options as `search`, appends the entry, and reports
its citation key. A missing file is created; existing entries and formatting
are preserved.

Duplicates with matching DOIs or database identifiers are skipped. Use `--key`
to choose your own citation key or resolve a key collision. Use `--dry-run` to
preview the entry without changing the file. Cancelling or a failed lookup
leaves the file unchanged.

### Journal abbreviations

```sh
bibi abbr inventiones mathematicae
# Invent. Math.

bibi abbr theory number journal
bibi abbr "ann. math."
```

`abbr` works offline. Search by words or word beginnings from the journal title
or abbreviation, in any order. Case, accents, and punctuation do not matter.
A unique match prints immediately; multiple matches open a picker.

### Other lookup commands

Search zbMATH Open directly:

```sh
bibi zb "zhang p-adic"
```

Search the free MR Lookup service by author, with an optional title and year:

```sh
bibi mr serre "a course in arithmetic" 1973
```

MR Lookup has more limited coverage than the full MathSciNet database. Use `-`
to skip the title when searching by author and year: `bibi mr serre - 1973`.

Find a mathematician's thesis through the Mathematics Genealogy Project:

```sh
bibi phd "carl gauss"
```

### BibTeX options

Choose a journal-name preference for any lookup:

```sh
bibi search "local fields" --journal full
bibi search "local fields" --journal short
```

`--journal source` is the default and uses the provider's journal name. Choose
`full` or `short` for the corresponding form, when available. Otherwise, the
original name is kept.

Warnings about incomplete entries do not prevent exporting or saving the BibTeX.

## Troubleshooting

BibTeX lookups require internet access. Temporary network failures are retried
automatically; press `Ctrl-C` to cancel.

Some zbMATH records cannot be exported because of license restrictions. If the
record has a DOI, try Crossref:

```sh
bibi search 10.2307/2306804 --discovery crossref --bib crossref
```

For more information about a lookup failure, add `--debug`, for example
`bibi search "local fields" --bib mr --debug`. Debug messages do not appear
in redirected BibTeX output. Set `NO_COLOR=1` to disable colours.

## Development

To build from a checkout, use the Go version declared in [`go.mod`](go.mod)
or newer:

```sh
go build -o bibi .
```

Run the local build with `./bibi`. Check changes with:

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

See the [bundled journal data notes](internal/journals/README.md) for catalog
refresh instructions.
