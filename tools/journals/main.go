// Command journals converts a serials CSV into the embedded journal catalog.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Each record stores abbreviation, title, translated title, and ISSN.
type record [4]string

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: go run ./tools/journals INPUT.csv OUTPUT.json.gz")
	}
	input, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer input.Close()
	records, err := parseCSV(input)
	if err != nil {
		return err
	}
	data, err := encodeCatalog(records)
	if err != nil {
		return err
	}
	if err := os.WriteFile(args[1], data, 0644); err != nil {
		return err
	}
	fmt.Printf("Converted %d journal records into %s (%d bytes)\n", len(records), args[1], len(data))
	return nil
}

func parseCSV(input io.Reader) ([]record, error) {
	reader := csv.NewReader(input)
	// Some publisher fields contain unescaped quotes and commas. Read the
	// fixed title columns from the front and ISSN from the end of each record.
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	wantHeader := []string{"Abbrev", "Full Title", "Trnsl. Title", "Publ.", "ISSN", "New", "Cover-to-cover", "Book Ser"}
	if len(header) != len(wantHeader) {
		return nil, fmt.Errorf("unexpected CSV header")
	}
	for i, name := range wantHeader {
		if strings.TrimSpace(strings.TrimPrefix(header[i], "\ufeff")) != name {
			return nil, fmt.Errorf("unexpected CSV column %d: %q", i+1, header[i])
		}
	}
	var records []record
	for row := 2; ; row++ {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", row, err)
		}
		if len(fields) < len(wantHeader) {
			return nil, fmt.Errorf("CSV row %d has %d columns, want at least %d", row, len(fields), len(wantHeader))
		}
		journal := record{strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]),
			strings.TrimSpace(fields[2]), strings.TrimSpace(fields[len(fields)-4])}
		if journal[0] == "" {
			return nil, fmt.Errorf("CSV row %d has no abbreviation", row)
		}
		records = append(records, journal)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV has no journal entries")
	}
	return records, nil
}

func encodeCatalog(records []record) ([]byte, error) {
	var output bytes.Buffer
	compressed, err := gzip.NewWriterLevel(&output, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if err := json.NewEncoder(compressed).Encode(records); err != nil {
		compressed.Close()
		return nil, err
	}
	if err := compressed.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
