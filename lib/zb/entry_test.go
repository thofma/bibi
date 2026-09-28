package zb

import (
	"encoding/json"
	"testing"
)

func TestParseToStructReturnsJSONError(t *testing.T) {
	response, err := ParseToStruct(`{"result":`)
	if err == nil {
		t.Fatal("ParseToStruct() error = nil, want JSON parse error")
	}
	if len(response.Result) != 0 {
		t.Fatalf("ParseToStruct() result count = %d, want 0 on parse error", len(response.Result))
	}
}

func TestParseToStructDecodesBookYear(t *testing.T) {
	response, err := ParseToStruct(`{
		"result": [{
			"source": {"book": [{"book_id": 7, "publisher": "MSP", "year": "2019"}]}
		}]
	}`)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	if len(response.Result) != 1 || len(response.Result[0].Source.Book) != 1 {
		t.Fatalf("ParseToStruct() decoded unexpected response: %+v", response)
	}
	if got := response.Result[0].Source.Book[0].Year; got != "2019" {
		t.Errorf("Book.Year = %q, want %q", got, "2019")
	}
}

func TestParseToStructAllowsObjectBiographicReferences(t *testing.T) {
	response, err := ParseToStruct(`{
		"result": [{
			"biographic_references": [{
				"aliases": [],
				"checked": "1",
				"codes": ["zariski.oscar"],
				"name": "Zariski, Oscar"
			}]
		}]
	}`)
	if err != nil {
		t.Fatalf("ParseToStruct() error = %v", err)
	}
	if len(response.Result) != 1 || len(response.Result[0].BiographicReferences) != 1 {
		t.Fatalf("unexpected biographic references: %+v", response.Result)
	}

	var author Author
	if err := json.Unmarshal(response.Result[0].BiographicReferences[0], &author); err != nil {
		t.Fatalf("biographic reference is not preserved JSON: %v", err)
	}
	if got, want := author.Name, "Zariski, Oscar"; got != want {
		t.Errorf("biographic reference author = %q, want %q", got, want)
	}
}
