package bibliography

import (
	"fmt"
	"strings"

	"github.com/nickng/bibtex"
)

// ParseBibTeX retains native field contents while accepting the capitalized
// month macros used by DOI services. The parser's built-in names are lowercase.
func ParseBibTeX(body []byte) (*bibtex.BibTex, error) {
	var source strings.Builder
	for _, month := range []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"} {
		capitalized := strings.ToUpper(month[:1]) + month[1:]
		fmt.Fprintf(&source, "@string{%s = %s}\n@string{%s = %s}\n", capitalized, month, strings.ToUpper(month), month)
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
