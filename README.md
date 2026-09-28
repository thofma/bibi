# bibi

`bibi` is a command-line tool that retrieves BibTeX for mathematical literature
from MR Lookup, zbMATH Open, and the Mathematics Genealogy Project.

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

Each command writes a single BibTeX entry to standard output. Errors and the
interactive result picker are written to standard error. Press `q` or `Ctrl-C`
to cancel a picker.

### zbMATH Open

```sh
bibi zb "hofmann zhang p-adic"
```

`zb` searches zbMATH Open and lets you choose among up to ten results.

### MR Lookup

```sh
bibi mr serre "a course in arithmetic" 1973
```

The title and year are optional. Use `-` to leave an earlier field empty, for
example `bibi mr serre - 1973`.

### Mathematics Genealogy Project

```sh
bibi phd "carl gauss"
```

`phd` searches the Mathematics Genealogy Project for a mathematician's thesis.

All lookups require network access to their respective services.

## Development

Run the repository checks with:

```sh
go test ./...
go vet ./...
go build ./...
gofmt -l .
```
