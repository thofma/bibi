package bibliography

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Identifier struct {
	Kind  string
	Value string
}

// ParseIdentifier accepts a whole DOI or arXiv locator, never a free-text query.
func ParseIdentifier(input string) (Identifier, error) {
	if id, ok := ArXivQuery(input); ok {
		return Identifier{Kind: "arxiv", Value: id}, nil
	}
	if doi, ok := DOIQuery(input); ok {
		return Identifier{Kind: "doi", Value: doi}, nil
	}
	return Identifier{}, fmt.Errorf("expected a DOI or arXiv identifier or URL; use bibi search for free text")
}

var arxivPattern = regexp.MustCompile(`^(?:([0-9]{4})\.([0-9]{4,5})|[a-z]+(?:[.-][a-z]+)*/([0-9]{7}))(?:v[1-9][0-9]*)?$`)
var arxivVersionPattern = regexp.MustCompile(`v[1-9][0-9]*$`)

// ArXivQuery normalizes modern and legacy IDs and abstract, PDF and HTML URLs.
// Explicit versions are retained; URL query strings and fragments are ignored.
func ArXivQuery(input string) (string, bool) {
	id := strings.TrimSpace(input)
	lower := strings.ToLower(id)
	if strings.HasPrefix(lower, "arxiv:") {
		id = strings.TrimSpace(id[len("arxiv:"):])
	} else {
		for _, host := range []string{"arxiv.org", "www.arxiv.org", "export.arxiv.org"} {
			if strings.HasPrefix(lower, host+"/") {
				id = "https://" + id
				break
			}
		}
		parsed, err := url.Parse(id)
		if err != nil {
			return "", false
		}
		if parsed.Host != "" || parsed.Scheme != "" {
			if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
				return "", false
			}
			switch strings.ToLower(parsed.Host) {
			case "arxiv.org", "www.arxiv.org", "export.arxiv.org":
			default:
				return "", false
			}
			path := strings.TrimPrefix(parsed.Path, "/")
			kind, rest, found := strings.Cut(path, "/")
			if !found || (kind != "abs" && kind != "pdf" && kind != "html") {
				return "", false
			}
			id = strings.TrimSuffix(rest, "/")
			if kind == "pdf" {
				id = strings.TrimSuffix(id, ".pdf")
			}
		}
	}
	id = strings.ToLower(id)
	match := arxivPattern.FindStringSubmatch(id)
	if match == nil {
		return "", false
	}
	date := match[1]
	if date != "" {
		// The modern scheme began in April 2007 and gained a fifth sequence
		// digit in January 2015. Dates in either scheme must have a real month.
		if date < "0704" || (date < "1501" && len(match[2]) != 4) || (date >= "1501" && len(match[2]) != 5) {
			return "", false
		}
	} else {
		date = match[3][:4]
	}
	month, _ := strconv.Atoi(date[2:])
	if month < 1 || month > 12 {
		return "", false
	}
	return id, true
}

func ArXivBaseID(id string) string {
	return arxivVersionPattern.ReplaceAllString(id, "")
}
