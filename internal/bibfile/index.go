// Package bibfile adds citations without reformatting existing bibliography text.
package bibfile

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type valuePart struct {
	text  string
	macro bool
}

type bibValue []valuePart

type indexedEntry struct {
	key    string
	fields map[string]bibValue
	ids    map[string]string
}

type fileIndex struct {
	entries []indexedEntry
	macros  map[string]macroValue
}

type macroValue struct {
	text string
	err  error
}

// The reader parses structure and value expressions, retaining the original
// bytes separately. Unknown macros in ordinary fields need no evaluation.
type reader struct {
	text string
	pos  int
}

func indexFile(data []byte) (fileIndex, error) {
	r := reader{text: string(data)}
	index := fileIndex{macros: make(map[string]macroValue)}
	keys := make(map[string]bool)
	for r.pos < len(r.text) {
		r.space()
		if r.pos >= len(r.text) {
			break
		}
		if r.text[r.pos] != '@' {
			r.pos++ // BibTeX permits commentary outside entries.
			continue
		}
		r.pos++
		kind := strings.ToLower(r.name())
		r.space()
		if kind == "" || r.pos >= len(r.text) || (r.text[r.pos] != '{' && r.text[r.pos] != '(') {
			switch kind {
			case "article", "book", "booklet", "inbook", "incollection", "inproceedings", "conference", "manual", "mastersthesis", "misc", "phdthesis", "proceedings", "techreport", "unpublished", "string", "preamble", "xdata", "set":
				return fileIndex{}, r.errorf("expected an opening delimiter after @%s", kind)
			}
			continue // An ordinary @ in commentary, e.g. an email address.
		}
		open := r.text[r.pos]
		close := byte('}')
		if open == '(' {
			close = ')'
		}
		if kind == "comment" {
			if err := r.comment(open, close); err != nil {
				return fileIndex{}, err
			}
			continue
		}
		r.pos++
		r.space()
		if kind == "preamble" {
			if _, err := r.value(close); err != nil {
				return fileIndex{}, err
			}
			r.space()
			r.take(',')
			if err := r.expect(close); err != nil {
				return fileIndex{}, err
			}
			continue
		}
		entry := indexedEntry{fields: make(map[string]bibValue)}
		if kind != "string" {
			start := r.pos
			for r.pos < len(r.text) && r.text[r.pos] != ',' && r.text[r.pos] != close && r.text[r.pos] != '%' {
				r.pos++
			}
			entry.key = strings.TrimSpace(r.text[start:r.pos])
			if err := ValidateKey(entry.key); err != nil {
				return fileIndex{}, r.errorf("%v", err)
			}
			if keys[strings.ToLower(entry.key)] {
				return fileIndex{}, r.errorf("citation key %q is used more than once (including case variants)", entry.key)
			}
			keys[strings.ToLower(entry.key)] = true
			r.space()
			if !r.take(close) {
				if err := r.expect(','); err != nil {
					return fileIndex{}, err
				}
			} else {
				index.entries = append(index.entries, entry)
				continue
			}
		}
		for {
			r.space()
			if r.take(close) {
				break
			}
			field := strings.ToLower(r.name())
			if field == "" {
				return fileIndex{}, r.errorf("expected a field name")
			}
			if err := r.expect('='); err != nil {
				return fileIndex{}, err
			}
			value, err := r.value(close)
			if err != nil {
				return fileIndex{}, err
			}
			if kind == "string" {
				text, err := resolveValue(value, index.macros)
				index.macros[field] = macroValue{text: text, err: err}
			} else {
				if _, exists := entry.fields[field]; exists {
					return fileIndex{}, r.errorf("entry %q repeats field %q", entry.key, field)
				}
				entry.fields[field] = value
			}
			r.space()
			if r.take(close) {
				break
			}
			if err := r.expect(','); err != nil {
				return fileIndex{}, err
			}
		}
		if kind != "string" {
			ids, err := entryIdentifiers(entry.fields, index.macros)
			if err != nil {
				return fileIndex{}, fmt.Errorf("entry %q: %w", entry.key, err)
			}
			entry.ids = ids
			index.entries = append(index.entries, entry)
		}
	}
	return index, nil
}

// ValidateKey accepts citation keys without BibTeX delimiters or whitespace.
func ValidateKey(key string) error {
	if key == "" || !utf8.ValidString(key) {
		return fmt.Errorf("citation key must be nonempty valid text")
	}
	for _, char := range key {
		if unicode.IsSpace(char) || unicode.IsControl(char) || strings.ContainsRune("{}(),=\"#%\\", char) {
			return fmt.Errorf("invalid citation key %q: whitespace and BibTeX delimiters are not allowed", key)
		}
	}
	return nil
}

func (r *reader) space() {
	for r.pos < len(r.text) {
		switch r.text[r.pos] {
		case ' ', '\t', '\r', '\n', '\f':
			r.pos++
		case '%':
			for r.pos < len(r.text) && r.text[r.pos] != '\n' {
				r.pos++
			}
		default:
			return
		}
	}
}

func (r *reader) name() string {
	start := r.pos
	for r.pos < len(r.text) {
		char := r.text[r.pos]
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("_:-.", rune(char))) {
			break
		}
		r.pos++
	}
	return r.text[start:r.pos]
}

func (r *reader) take(char byte) bool {
	if r.pos < len(r.text) && r.text[r.pos] == char {
		r.pos++
		return true
	}
	return false
}

func (r *reader) expect(char byte) error {
	r.space()
	if !r.take(char) {
		return r.errorf("expected %q", char)
	}
	return nil
}

func (r *reader) errorf(format string, args ...any) error {
	line := strings.Count(r.text[:r.pos], "\n") + 1
	return fmt.Errorf("line %d: %s", line, fmt.Sprintf(format, args...))
}

func (r *reader) value(close byte) (bibValue, error) {
	var parts bibValue
	for {
		r.space()
		if r.pos >= len(r.text) {
			return nil, r.errorf("unfinished field value")
		}
		part := valuePart{}
		var err error
		switch r.text[r.pos] {
		case '{':
			part.text, err = r.delimited(false)
		case '"':
			part.text, err = r.delimited(true)
		default:
			start := r.pos
			part.text = r.name()
			if r.pos == start || r.text[start] == close {
				return nil, r.errorf("expected a field value")
			}
			part.macro = !decimal(part.text)
		}
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
		r.space()
		if !r.take('#') {
			return parts, nil
		}
	}
}

func (r *reader) delimited(quoted bool) (string, error) {
	r.pos++
	start, depth := r.pos, 0
	for r.pos < len(r.text) {
		char := r.text[r.pos]
		if char == '\\' {
			r.pos++
			if r.pos < len(r.text) {
				r.pos++
			}
			continue
		}
		if quoted && char == '"' && depth == 0 || !quoted && char == '}' && depth == 0 {
			value := r.text[start:r.pos]
			r.pos++
			return value, nil
		}
		if char == '{' {
			depth++
		} else if char == '}' {
			depth--
			if depth < 0 {
				return "", r.errorf("unbalanced braces in quoted value")
			}
		}
		r.pos++
	}
	return "", r.errorf("unfinished braced or quoted value")
}

func (r *reader) comment(open, close byte) error {
	r.pos++
	depth := 1
	for r.pos < len(r.text) {
		char := r.text[r.pos]
		r.pos++
		if char == '\\' && r.pos < len(r.text) {
			r.pos++
		} else if char == open {
			depth++
		} else if char == close {
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
	return r.errorf("unfinished comment")
}

func decimal(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
