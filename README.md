<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/bibi-mark-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="assets/bibi-mark-light.svg">
  <img alt="bibi" src="assets/bibi-mark-light.svg" width="88">
</picture>

# bibi

`bibi` is a command-line tool that retrieves BibTeX for mathematical literature
from MathSciNet's limited free MRTools search, [zbMATH Open](https://zbmath.org/),
and the [Mathematics Genealogy Project](https://www.genealogy.math.ndsu.nodak.edu/).

## Install

Install the Go version declared in [`go.mod`](go.mod), then build from a checkout:

```sh
go build -o bibi .
```

Or install it into your Go binary directory:

```sh
go install .
```

## Usage

Each command writes a single BibTeX entry to standard output. Search progress,
errors, and the interactive result picker are written to standard error. Press
`q` or `Ctrl-C` to cancel a picker.

### zbMATH Open

```sh
bibi zb "hofmann zhang p-adic"
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
