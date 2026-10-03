package bibliography

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseBibTeXFullMonthMacros(t *testing.T) {
	for _, month := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
		for _, macro := range []string{month, strings.ToLower(month), strings.ToUpper(month)} {
			t.Run(macro, func(t *testing.T) {
				source := fmt.Sprintf(`@article{Ore1952,title={The General Chinese Remainder Theorem},month=%s}`, macro)
				parsed, err := ParseBibTeX([]byte(source))
				if err != nil || len(parsed.Entries) != 1 || parsed.Entries[0].Fields["month"].String() != month {
					t.Fatalf("month %s: parsed=%+v error=%v", macro, parsed, err)
				}
			})
		}
	}
}

func TestParseBibTeXRecoversAfterMalformedEntries(t *testing.T) {
	for _, malformed := range []string{
		"@article{broken, title={unfinished",
		`@article{broken, title="unfinished`,
		"@article{broken, author=",
	} {
		if _, err := ParseBibTeX([]byte(malformed)); err == nil {
			t.Fatalf("accepted malformed BibTeX: %s", malformed)
		}
		parsed, err := ParseBibTeX([]byte(`@article{Valid, title={{ABC} and $p$-adic fields}, author={Doe, Jane}, year=2020, month=Jan}`))
		if err != nil {
			t.Fatalf("valid entry failed after %q: %v", malformed, err)
		}
		if len(parsed.Entries) != 1 || parsed.Entries[0].CiteName != "Valid" || parsed.Entries[0].Fields["title"].String() != "{ABC} and $p$-adic fields" || parsed.Entries[0].Fields["month"].String() != "January" {
			t.Fatalf("entry changed after malformed input: %+v", parsed)
		}
	}
}
