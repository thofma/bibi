package bibfile

import (
	"reflect"
	"strings"
	"testing"
)

func TestIndexRealisticBibliography(t *testing.T) {
	input := `% User notes and a@example.org; ignored @article{fake, doi={10.1000/fake}}
@comment{Unused record: @book{ignored, doi={10.1000/unused}}}
@preamble("\\newcommand{\\Q}{\\mathbb{Q}}" # { })
@string{prefix = "https://doi.org/10.1000/"}
@string(suffix = {a\_b}, identifier = prefix # suffix)
@article(Original,
  TITLE = {An {É}tude of {\(p\)}-adic fields},
  author = "Doe, Jane and {Research {Group}}",
  journal = externalJournalMacro,
  month = jan,
  DOI = identifier,
  MRNUMBER = {MR0001234},
  zbmath = 0005678,
  zbl = {Zbl 0012.00345}, % a comment after a field
)
@string{prefix = {10.1000/second}}
@misc{Second, DOI=prefix, title={A second edition}, note="A \"quote\""}
@misc{Preprint, archivePrefix={arXiv}, eprint={arXiv:2401.12345v2}}
@misc{Empty}
`
	index, err := indexFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(index.entries) != 4 {
		t.Fatalf("indexed entries = %+v", index.entries)
	}
	want := map[string]string{"doi": "10.1000/a_b", "mrnumber": "1234", "zbmath": "5678", "zbl": "12.345"}
	if !reflect.DeepEqual(index.entries[0].ids, want) || index.entries[1].ids["doi"] != "10.1000/second" || index.entries[2].ids["arxiv"] != "2401.12345v2" {
		t.Fatalf("identifiers = %+v / %+v / %+v", index.entries[0].ids, index.entries[1].ids, index.entries[2].ids)
	}
}

func TestIndexFailsClosedForAmbiguousFiles(t *testing.T) {
	for _, input := range []string{
		`@article key, title={missing opener}`,
		`@book`,
		`@article{key, title={unfinished}`,
		`@article{key, title="unfinished}`,
		`@article{key, doi=unknownIdentifier}`,
		`@string{x=undefined} @article{key, doi=x}`,
		`@article{key, doi={not a DOI}}`,
		`@article{key, MRNUMBER={unknown}}`,
		`@article{key, zbmath={xyz}}`,
		`@article{key, zbl={12345}}`,
		`@article{key, doi={10.1000/x}, DOI={10.1000/y}}`,
		`@misc{Key} @misc{key}`,
		`@misc{bad key}`,
		`@misc{key, title={okay} author={missing comma}}`,
		`@comment{unfinished`,
		`@preamble{{value} # }`,
		`@misc{key, title="unbalanced }"}`,
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := indexFile([]byte(input)); err == nil {
				t.Fatalf("accepted unsafe bibliography %s", input)
			}
		})
	}
}

func TestDuplicateIdentityAndConflicts(t *testing.T) {
	for _, test := range []struct {
		name, existing, incoming, errorText string
		keys                                []string
	}{
		{"DOI resolver and TeX", `@misc{old, DOI={https://doi.org/10.1000/A\_B}}`, `@misc{new, doi={10.1000/a_b}}`, "", []string{"old"}},
		{"MR prefix", `@misc{old, MRNUMBER={MR001234}}`, `@misc{new, mrnumber=1234}`, "", []string{"old"}},
		{"zb identifier", `@misc{old, zbmath={0001234}}`, `@misc{new, zbmath=1234}`, "", []string{"old"}},
		{"Zbl number", `@misc{old, zbl={0123.00456}}`, `@misc{new, zbl={123.456}}`, "", []string{"old"}},
		{"arXiv identifier", `@misc{old, eprinttype={arxiv}, eprint={2401.12345v2}}`, `@misc{new, archiveprefix={arXiv}, eprint={arXiv:2401.12345v2}}`, "", []string{"old"}},
		{"arXiv versions differ", `@misc{old, archiveprefix={arXiv}, eprint={2401.12345v1}}`, `@misc{new, archiveprefix={arXiv}, eprint={2401.12345v2}}`, "", nil},
		{"similar metadata is insufficient", `@book{old,title={Local fields},author={Serre},year=1979}`, `@book{new,title={Local fields},author={Serre},year=1979}`, "", nil},
		{"same key without identifiers", `@misc{Key,title={Same}}`, `@misc{key,title={Same}}`, "--key", nil},
		{"same key different DOI", `@misc{key,doi={10.1000/a}}`, `@misc{key,doi={10.1000/b}}`, "--key", nil},
		{"one equal ID does not erase a conflict", `@misc{old,doi={10.1000/a},mrnumber=123}`, `@misc{new,doi={10.1000/a},mrnumber=456}`, "conflicting identifiers", nil},
		{"multiple existing keys", `@misc{first,doi={10.1000/a}} @misc{second,doi={10.1000/a}}`, `@misc{new,doi={10.1000/a}}`, "", []string{"first", "second"}},
		{"duplicate with conflicting key", `@misc{first,doi={10.1000/a}} @misc{new,doi={10.1000/b}}`, `@misc{new,doi={10.1000/a}}`, "--key", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			existing, err := indexFile([]byte(test.existing))
			if err != nil {
				t.Fatal(err)
			}
			incoming, err := indexFile([]byte(test.incoming))
			if err != nil {
				t.Fatal(err)
			}
			keys, err := duplicateKeys(existing, incoming.entries[0])
			if test.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), test.errorText) {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil || !reflect.DeepEqual(keys, test.keys) {
				t.Fatalf("keys = %v, error = %v", keys, err)
			}
		})
	}
}

func FuzzIndexFile(f *testing.F) {
	for _, seed := range []string{"", `@misc{key,title={nested {text}}}`, `@string{a="x"} @misc{k,journal=a}`, `@misc(k,doi={10.1000/x})`, `@comment{hello}`, "\x00\xff"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = indexFile(data) })
}
