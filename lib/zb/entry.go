package zb

import (
	"encoding/json"
)

// Define all necessary structs to match the JSON data

type Author struct {
	Aliases []string `json:"aliases"`
	Checked string   `json:"checked"`
	Codes   []string `json:"codes"`
	Name    string   `json:"name"`
}

type Reviewer struct {
	AuthorCode string `json:"author_code"`
	ReviewerID string `json:"reviewer_id"`
	Name       string `json:"name"`
	Sign       string `json:"sign"`
}

type EditorialContribution struct {
	Language         string   `json:"language"`
	Reviewer         Reviewer `json:"reviewer"`
	Text             string   `json:"text"`
	ContributionType string   `json:"contribution_type"`
}

type DocumentType struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type Link struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	URL        string `json:"url"`
}

type MSC struct {
	Code   string `json:"code"`
	Scheme string `json:"scheme"`
	Text   string `json:"text"`
}

type ZBMath struct {
	AuthorCodes []string `json:"author_codes"`
	DocumentID  int      `json:"document_id"`
	MSC         []string `json:"msc"`
	Prefix      string   `json:"prefix"`
	SeriesID    int      `json:"series_id"`
	Year        string   `json:"year"`
}

type Reference struct {
	DOI      string `json:"doi"`
	Position string `json:"position"`
	Text     string `json:"text"`
	ZBMath   ZBMath `json:"zbmath"`
}

type ISBN struct {
	Number string `json:"number"`
	Type   string `json:"type"`
}

type Book struct {
	BookID    int    `json:"book_id"`
	ISBN      []ISBN `json:"isbn"`
	Publisher string `json:"publisher"`
	Title     string `json:"title"`
	Year      string `json:"year"`
}

type Source struct {
	Book   []Book   `json:"book"`
	Pages  string   `json:"pages"`
	Series []Series `json:"series"`
	Source string   `json:"source"`
}

type Series struct {
	Acronym    string `json:"acronym"`
	ISSN       []ISSN `json:"issn"`
	Issue      string `json:"issue"`
	IssueID    int    `json:"issue_id"`
	Publisher  string `json:"publisher"`
	SeriesID   int    `json:"series_id"`
	ShortTitle string `json:"short_title"`
	Title      string `json:"title"`
	Volume     string `json:"volume"`
	Year       string `json:"year"`
}

type ISSN struct {
	Number string `json:"number"`
	Type   string `json:"type"`
}

type Title struct {
	Original string `json:"original"`
	Subtitle string `json:"subtitle"`
	Title    string `json:"title"`
	Addition string `json:"addition"`
}

type Language struct {
	Languages []string `json:"languages"`
	Addition  []string `json:"addition"`
}

type Item struct {
	// BiographicReferences is ancillary metadata whose entries vary between
	// strings and author-like objects in the zbMath API.
	BiographicReferences   []json.RawMessage       `json:"biographic_references"`
	Contributors           Contributors            `json:"contributors"`
	Database               string                  `json:"database"`
	Datestamp              string                  `json:"datestamp"`
	DocumentType           DocumentType            `json:"document_type"`
	EditorialContributions []EditorialContribution `json:"editorial_contributions"`
	ID                     int                     `json:"id"`
	Identifier             string                  `json:"identifier"`
	Keywords               []string                `json:"keywords"`
	Language               Language                `json:"language"`
	License                []string                `json:"license"`
	Links                  []Link                  `json:"links"`
	MSC                    []MSC                   `json:"msc"`
	References             []Reference             `json:"references"`
	Source                 Source                  `json:"source"`
	States                 [][]string              `json:"states"`
	Title                  Title                   `json:"title"`
	Year                   string                  `json:"year"`
	ZBMathURL              string                  `json:"zbmath_url"`
}

type Contributors struct {
	Authors          []Author `json:"authors"`
	AuthorReferences []string `json:"author_references"`
	Editors          []Author `json:"editors"`
}

type Response struct {
	Result []Item `json:"result"`
	Status Status `json:"status"`
}

type Status struct {
	Execution          string  `json:"execution"`
	ExecutionBool      bool    `json:"execution_bool"`
	InternalCode       string  `json:"internal_code"`
	LastID             *string `json:"last_id"`
	NrTotalResults     int     `json:"nr_total_results"`
	NrRequestResults   int     `json:"nr_request_results"`
	QueryExecutionTime float64 `json:"query_execution_time_in_seconds"`
	StatusCode         int     `json:"status_code"`
	TimeStamp          string  `json:"time_stamp"`
}

func ParseToStruct(body string) (Response, error) {
	var response Response
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return Response{}, err
	}
	return response, nil
}

// @article{zbMATH06617923,
//  author = {Hofmann, Tommy and Zhang, Yinan},
//  title = {Valuations of {{\(p\)}}-adic regulators of cyclic cubic fields},
//  fjournal = {Journal of Number Theory},
//  journal = {J. Number Theory},
//  issn = {0022-314X},
//  volume = {169},
//  pages = {86--102},
//  year = {2016},
//  language = {English},
//  doi = {10.1016/j.jnt.2016.05.016},
//  keywords = {11Y40,11R16,11R27},
//  zbMATH = {6617923},
//  Zbl = {1409.11141}
// }
//
// @article {MR3531231,
//    AUTHOR = {Hofmann, Tommy and Zhang, Yinan},
//     TITLE = {Valuations of {$p$}-adic regulators of cyclic cubic fields},
//   JOURNAL = {J. Number Theory},
//  FJOURNAL = {Journal of Number Theory},
//    VOLUME = {169},
//      YEAR = {2016},
//     PAGES = {86--102},
//      ISSN = {0022-314X},
//   MRCLASS = {11Y40 (11K41 11R16 11R27)},
//  MRNUMBER = {3531231},
//MRREVIEWER = {Ken Yamamura},
//       DOI = {10.1016/j.jnt.2016.05.016},
//       URL = {https://doi.org/10.1016/j.jnt.2016.05.016},
//}

// @incollection{zbMATH07721132,
// author = {Hofmann, Tommy and Zhang, Yinan},
// title = {Cyclic extensions of prime degree and their {{\(p\)}}-adic regulators},
// booktitle = {ANTS XIII. Proceedings of the thirteenth algorithmic number theory symposium, University of Wisconsin-Madison, WI, USA, July 16--20, 2018},
// isbn = {978-1-935107-02-6; 978-1-935107-03-3},
// pages = {311--323},
// year = {2019},
// publisher = {Berkeley, CA: Mathematical Sciences Publishers (MSP)},
// language = {English},
// doi = {10.2140/obs.2019.2.311},
// keywords = {11Y40,11K41,11R20,11R27},
// zbMATH = {7721132},
// Zbl = {1531.11123}
//}
