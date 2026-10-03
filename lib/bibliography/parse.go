package bibliography

import (
	"fmt"
	"strings"

	"github.com/nickng/bibtex"
)

// ParseBibTeX retains native field contents while accepting month abbreviations
// and full names used by DOI services. The parser's built-in macros are lowercase
// three-letter abbreviations.
func ParseBibTeX(body []byte) (*bibtex.BibTex, error) {
	var source strings.Builder
	for _, month := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
		abbreviation := strings.ToLower(month[:3])
		defined := map[string]bool{abbreviation: true}
		for _, alias := range []string{month[:3], strings.ToUpper(abbreviation), month, strings.ToLower(month), strings.ToUpper(month)} {
			if !defined[alias] {
				fmt.Fprintf(&source, "@string{%s = %s}\n", alias, abbreviation)
				defined[alias] = true
			}
		}
	}
	source.Write(body)
	parsed, err := bibtex.Parse(strings.NewReader(source.String()))
	if err != nil {
		// nickng/bibtex keeps its field lexer mode globally and leaves it set
		// after some syntax errors. Feed a closing delimiter to clear that mode
		// before the next batch entry. This deliberately invalid reset input
		// produces no citation; its parse error is irrelevant.
		_, _ = bibtex.Parse(strings.NewReader("@}"))
	}
	return parsed, err
}
