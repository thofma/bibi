# Bundled journal catalog

`catalog.json.gz` contains 2,385 journal and book-series records. The snapshot
was converted on 2026-10-03. The compressed catalog replaces the original CSV.

SHA-256: `575fa9ddafac8a9ffc62f96bc074a1c5fdd4fee0598394ce4cad5f365eeb3e01`

The file is a gzip-compressed JSON array. Each record is a four-element array
containing its abbreviation, full title, translated title, and ISSN, in that
order. Fields retain their original spelling and TeX markup; surrounding
whitespace is trimmed. Some records have no full title or ISSN. Publisher and
status columns are omitted because the command does not use them.

Go embeds the compressed catalog in the executable. `bibi abbr` needs neither
network access nor an external data file.

To refresh it from a downloaded serials CSV, run from the repository root:

```sh
go run ./tools/journals /path/to/serials.csv internal/journals/catalog.json.gz
shasum -a 256 internal/journals/catalog.json.gz
go test ./tools/journals ./internal/journals ./cmd
```

The converter tolerates malformed quoting in publisher fields. It reads the
title columns from the beginning and ISSN from the end of each CSV record.
Compression is deterministic and does not include a filename or timestamp.

When refreshing, update the date, record count, and checksum here, and the
record count and parsed-fields checksum in `TestBundledSnapshot`. Refreshing
is a development step; the command never downloads data at runtime.
