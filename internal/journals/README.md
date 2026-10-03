# Bundled AMS serials list

`annser.csv` is an unmodified snapshot of the
[AMS MR Serials Abbreviations List](https://mathscinet.ams.org/msnhtml/annser.csv),
downloaded on 2026-10-03. It contains 2,385 records, including book series.

SHA-256: `17c685e138f3a2aefbb3e62579d7740b757aa70435389bc1cf7e735ecdc4979e`

Go embeds the CSV in the executable, so `bibi abbr` needs neither network access
nor an external data file. Abbreviations are returned with the AMS spelling,
including any TeX markup. Some source records have no full title or ISSN.

The CSV parser tolerates the source's malformed quoting in one publisher field.
It reads the title columns from the beginning and the ISSN and flags from the
end of each record, so a stray comma in the publisher does not shift the ISSN.

To refresh the snapshot, run from the repository root:

```sh
curl --fail --location --proto '=https' \
  --output internal/journals/annser.csv \
  https://mathscinet.ams.org/msnhtml/annser.csv
shasum -a 256 internal/journals/annser.csv
go test ./internal/journals ./cmd
```

Update the date, record count and checksum here, the date in the main README,
and the snapshot record count in the catalog test. Refreshing is a development
step; the command never downloads data at runtime.
