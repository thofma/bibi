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
and the [Mathematics Genealogy Project](https://www.genealogy.math.ndsu.nodak.edu/).

## Table of Contents

- [Install](#install)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [zbMATH Open](#zbmath-open)
  - [MR Lookup](#mr-lookup)
  - [Mathematics Genealogy Project](#mathematics-genealogy-project)
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
bibi zb "zhang p-adic"
```

If the search returns more than one match, choose a result when prompted. `bibi`
prints its BibTeX entry to standard output.

## Usage

Each command writes a single BibTeX entry to standard output. Search progress,
errors, and the interactive result picker are written to standard error. Press
`q` or `Ctrl-C` to cancel a picker.

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

## Development

Run the repository checks with:

```sh
go test ./...
go vet ./...
go build ./...
gofmt -l .
```
