package zb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func restrictedRecord(t *testing.T) Item {
	t.Helper()
	body, err := os.ReadFile("testdata/restricted-record.json")
	if err != nil {
		t.Fatal(err)
	}
	response, err := ParseToStruct(string(body))
	if err != nil || len(response.Result) != 1 {
		t.Fatalf("restricted fixture: response=%+v error=%v", response, err)
	}
	return response.Result[0]
}

func TestRestrictedRecordKeepsAvailableDiscoveryMetadata(t *testing.T) {
	item := restrictedRecord(t)
	work := ItemWork(item)
	if work.Title != "" || len(work.Authors) != 0 || work.Notes != "" {
		t.Fatalf("license notices became citation metadata: %+v", work)
	}
	if work.Year != "1952" || work.Venue != "American Mathematical Monthly" || work.Type != "journal article" ||
		work.DOI != "10.2307/2306804" || work.IDs["zb"] != "3077854" || work.IDs["zbl"] != "0049.31002" {
		t.Fatalf("available metadata was lost: %+v", work)
	}
	if work.Label() != "Metadata restricted, 1952, DOI: 10.2307/2306804" {
		t.Fatalf("restricted label = %q", work.Label())
	}
	details := work.Details()
	if strings.Contains(details, licensePlaceholder) || strings.Count(details, "license restrictions") != 1 ||
		!strings.Contains(details, "bibi search 10.2307/2306804 --discovery crossref --bib crossref") ||
		!strings.Contains(details, "Venue: American Mathematical Monthly") {
		t.Fatalf("restricted details = %q", details)
	}
	if strings.Contains(work.Query(), "license") || strings.Contains(work.Query(), "crossref") {
		t.Fatalf("restriction notice leaked into provider query: %q", work.Query())
	}
	// Discovery must not change the cached source record or its contributors.
	if item.Title.Title != licensePlaceholder || item.Contributors.Authors[0].Name != licensePlaceholder {
		t.Fatal("discovery mutated native metadata")
	}
}

func TestRestrictedRecordCannotBeExported(t *testing.T) {
	entry, err := ItemToBibEntry(restrictedRecord(t))
	if entry != nil || err == nil || !strings.Contains(err.Error(), "cannot export zbMATH record 3077854") ||
		!strings.Contains(err.Error(), "bibi search 10.2307/2306804 --discovery crossref --bib crossref") {
		t.Fatalf("restricted export: entry=%+v error=%v", entry, err)
	}
}

func TestRestrictedRecordWithoutDOIRetainsDatabaseIdentifier(t *testing.T) {
	item := restrictedRecord(t)
	item.Links = nil
	work := ItemWork(item)
	if work.Label() != "Metadata restricted, 1952, zbMATH: 3077854" ||
		!strings.Contains(work.MetadataNotice, "Try your search with --discovery crossref --bib crossref") {
		t.Fatalf("restricted record without DOI = %+v", work)
	}
}

func TestRestrictedRecordsDoNotInterruptPagingOrOtherExports(t *testing.T) {
	restricted := restrictedRecord(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		item := restricted
		if r.URL.Query().Get("page") == "1" {
			item = Item{ID: 42, DocumentType: DocumentType{Code: "j"}, Title: Title{Title: "Accessible work"}}
		}
		if err := json.NewEncoder(w).Encode(Response{Result: []Item{item}, Status: Status{NrTotalResults: 2}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	backend := &Backend{}
	backend.SetPageSize(1)
	first, err := backend.SearchPage(context.Background(), "number theory", "")
	if err != nil || len(first.Works) != 1 || first.NextToken != "1" || first.Total != 2 {
		t.Fatalf("restricted first page: %+v error=%v", first, err)
	}
	second, err := backend.SearchPage(context.Background(), "number theory", first.NextToken)
	if err != nil || len(second.Works) != 1 || second.NextToken != "" {
		t.Fatalf("second page: %+v error=%v", second, err)
	}
	if records, err := backend.BibTeX(first.Works[0]); err == nil || len(records) != 0 || !strings.Contains(err.Error(), "license restrictions") {
		t.Fatalf("restricted cached export: %+v error=%v", records, err)
	}
	records, err := backend.BibTeX(second.Works[0])
	if err != nil || len(records) != 1 || records[0].Title != "Accessible work" || requests != 2 {
		t.Fatalf("accessible cached export: %+v requests=%d error=%v", records, requests, err)
	}
}

func TestRestrictedAncillaryTextDoesNotBlockAccessibleCitation(t *testing.T) {
	item := Item{ID: 42, DocumentType: DocumentType{Code: "j"}, Title: Title{Title: "Accessible work", Original: licensePlaceholder},
		Contributors:           Contributors{Authors: []Author{{Name: "Doe, Jane"}}},
		Source:                 Source{Source: licensePlaceholder, Series: []Series{{Title: "Full journal", ShortTitle: "J."}}},
		EditorialContributions: []EditorialContribution{{Text: licensePlaceholder}},
		References:             []Reference{{Text: licensePlaceholder, DOI: licensePlaceholder}}}
	entry, err := ItemToBibEntry(item)
	if err != nil || entry.Fields["author"].String() != "Doe, Jane" || entry.Fields["journal"].String() != "J." {
		t.Fatalf("accessible citation: entry=%+v error=%v", entry, err)
	}
	work := ItemWork(item)
	if strings.Contains(work.Details(), licensePlaceholder) || work.Title != "Accessible work" || work.Venue != "Full journal" {
		t.Fatalf("accessible discovery metadata: %+v", work)
	}
	item.Title.Original = ""
	item.Source.Source = "Full journal"
	if notice := ItemWork(item).MetadataNotice; notice != "" {
		t.Fatalf("restricted review or reference affected citation availability: %q", notice)
	}
}
