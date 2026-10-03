package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const csvHeader = "Abbrev,Full Title,Trnsl. Title,Publ.,ISSN,New,Cover-to-cover,Book Ser\n"

func TestParseCSVPreservesFields(t *testing.T) {
	input := "\ufeff" + csvHeader +
		"\"Rev. Études\",\"Revue, des Études\",\"Review of Studies\",\"Publisher, City\",1234-5678,N,Y,N\n" +
		"\"Dopov.\",\"Dopovidi\",\"\",\"Vidavn. Dim \"Akademperiodika\", Kiev.\",\"1025-6415\",\"N\",\"N\",\"N\"\n" +
		" Abbr. ,,,,,N,N,Y\n"
	got, err := parseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []record{{"Rev. Études", "Revue, des Études", "Review of Studies", "1234-5678"},
		{"Dopov.", "Dopovidi", "", "1025-6415"}, {"Abbr.", "", "", ""}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %+v, want %+v", got, want)
	}
}

func TestParseCSVRejectsBrokenInput(t *testing.T) {
	for _, input := range []string{"", "wrong,columns\n", csvHeader, csvHeader + "J.,Journal\n", csvHeader + ",Journal,,,1234-5678,N,N,N\n"} {
		if _, err := parseCSV(strings.NewReader(input)); err == nil {
			t.Errorf("parseCSV(%q) succeeded, want invalid catalog", input)
		}
	}
}

func TestConversionIsLosslessAndReproducible(t *testing.T) {
	records := []record{{"Rev. Études", "Revue, des Études", "Review of Studies", "1234-5678"}, {"Abbr.", "", "", ""}}
	first, err := encodeCatalog(records)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodeCatalog(records)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("conversion produced different compressed bytes for identical records")
	}
	reader, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []record
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, records) {
		t.Fatalf("decoded = %+v, want %+v", decoded, records)
	}
}

func TestFailedConversionKeepsExistingCatalog(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.csv")
	output := filepath.Join(dir, "catalog.json.gz")
	if err := os.WriteFile(input, []byte("invalid input"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("previous catalog"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{input, output}); err == nil {
		t.Fatal("invalid input was converted")
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "previous catalog" {
		t.Fatalf("existing catalog changed: data=%q err=%v", data, err)
	}
}
