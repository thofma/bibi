package bibliography

import "testing"

func TestParseIdentifier(t *testing.T) {
	for _, test := range []struct {
		input, kind, value string
	}{
		{" 2301.12345 ", "arxiv", "2301.12345"},
		{"arXiv:2301.12345v2", "arxiv", "2301.12345v2"},
		{"https://arxiv.org/abs/2301.12345v2", "arxiv", "2301.12345v2"},
		{"https://arxiv.org/pdf/2301.12345v2.pdf?download=1#page=5", "arxiv", "2301.12345v2"},
		{"https://arxiv.org/html/2301.12345v2", "arxiv", "2301.12345v2"},
		{"arxiv.org/pdf/2301.12345", "arxiv", "2301.12345"},
		{"http://export.arxiv.org/abs/math/0303109v1", "arxiv", "math/0303109v1"},
		{"https://www.arxiv.org/pdf/hep-th/9901001.pdf", "arxiv", "hep-th/9901001"},
		{"math/0303109", "arxiv", "math/0303109"},
		{"arXiv:0704.0001v1", "arxiv", "0704.0001v1"},
		{"10.1000/ABC_def", "doi", "10.1000/abc_def"},
		{" DOI:10.1000/ABC_def ", "doi", "10.1000/abc_def"},
		{"http://dx.doi.org/10.1000/ABC%5Fdef", "doi", "10.1000/abc_def"},
		{"https://doi.org/10.1000/ABC_def?tracking=1#section", "doi", "10.1000/abc_def"},
		{"10.48550/arXiv.2301.12345", "doi", "10.48550/arxiv.2301.12345"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseIdentifier(test.input)
			if err != nil || got.Kind != test.kind || got.Value != test.value {
				t.Fatalf("ParseIdentifier = %+v, %v; want %s %s", got, err, test.kind, test.value)
			}
		})
	}
}

func TestParseIdentifierRejectsNonLocators(t *testing.T) {
	for _, input := range []string{
		"", "serre local fields", "paper 2301.12345", "10.1000/has space",
		"https://example.org/abs/2301.12345", "https://arxiv.org.evil/abs/2301.12345",
		"https://arxiv.org/search/2301.12345", "https://user@arxiv.org/abs/2301.12345",
		"ftp://arxiv.org/abs/2301.12345", "https://arxiv.org/abs/2301.12345/extra",
		"2301.12345v0", "2301.12345v", "2301.1234", "0704.12345", "0703.1234",
		"2334.12345", "math/0334999", "https://doi.org/", "arXiv:",
	} {
		if got, err := ParseIdentifier(input); err == nil {
			t.Errorf("accepted %q as %+v", input, got)
		}
	}
}
