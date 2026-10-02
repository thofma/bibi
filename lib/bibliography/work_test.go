package bibliography

import "testing"

func TestDOIQuery(t *testing.T) {
	for _, input := range []string{
		"10.1000/ABC_def", " doi:10.1000/ABC_def ",
		"https://doi.org/10.1000/ABC_def", "http://dx.doi.org/10.1000/ABC%5Fdef",
		`10.1000/ABC\_def`,
	} {
		doi, ok := DOIQuery(input)
		if !ok || doi != "10.1000/abc_def" {
			t.Errorf("DOIQuery(%q) = %q, %v", input, doi, ok)
		}
	}
	for _, query := range []string{"anything can go in here 1999", "serre 10.1000/example", "", "1999"} {
		if _, ok := DOIQuery(query); ok {
			t.Errorf("DOIQuery(%q) interpreted free text as a DOI", query)
		}
	}
}

func TestMatchIdentifiers(t *testing.T) {
	work := Work{DOI: "https://doi.org/10.1000/ABC", IDs: map[string]string{"zb": "1"}}
	for _, test := range []struct {
		name       string
		candidate  Work
		exact      bool
		compatible bool
	}{
		{"DOI", Work{DOI: "doi:10.1000/abc"}, true, true},
		{"identifier", Work{IDs: map[string]string{"zb": "1"}}, true, true},
		{"conflicting DOI vetoes identifier", Work{DOI: "10.1000/other", IDs: map[string]string{"zb": "1"}}, false, false},
		{"missing identifiers remain compatible", Work{}, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			exact, compatible := Match(work, test.candidate, "zb")
			if exact != test.exact || compatible != test.compatible {
				t.Fatalf("Match() = %v, %v, want %v, %v", exact, compatible, test.exact, test.compatible)
			}
		})
	}
}
