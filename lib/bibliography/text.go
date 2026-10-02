package bibliography

import "strings"

// EscapeTeXText makes text from metadata safe in LaTeX while retaining Unicode,
// existing TeX commands and their arguments, and dollar or LaTeX-delimited math.
// Use it for generated text fields, never to re-encode native BibTeX exports.
func EscapeTeXText(text string) string {
	var result strings.Builder
	mathEnd := ""
	for i := 0; i < len(text); {
		if mathEnd != "" {
			if strings.HasPrefix(text[i:], mathEnd) {
				result.WriteString(mathEnd)
				i += len(mathEnd)
				mathEnd = ""
			} else if text[i] == '\\' && i+1 < len(text) {
				result.WriteString(text[i : i+2])
				i += 2
			} else {
				result.WriteByte(text[i])
				i++
			}
			continue
		}
		if text[i] == '$' {
			mathEnd = "$"
			if strings.HasPrefix(text[i:], "$$") {
				mathEnd = "$$"
			}
			result.WriteString(mathEnd)
			i += len(mathEnd)
			continue
		}
		if text[i] == '\\' && i+1 < len(text) {
			if text[i+1] == '(' || text[i+1] == '[' {
				mathEnd = `\)`
				if text[i+1] == '[' {
					mathEnd = `\]`
				}
				result.WriteString(text[i : i+2])
				i += 2
				continue
			}
			end := i + 2
			for end < len(text) && isTeXLetter(text[end]) {
				end++
			}
			// TeX arguments are already encoded by the source; do not rewrite them.
			for end < len(text) && text[end] == '{' {
				groupEnd := braceGroupEnd(text, end)
				if groupEnd < 0 {
					break
				}
				end = groupEnd
			}
			result.WriteString(text[i:end])
			i = end
			continue
		}
		switch text[i] {
		case '&', '%', '#', '_':
			result.WriteByte('\\')
			result.WriteByte(text[i])
		case '~':
			result.WriteString(`\textasciitilde{}`)
		case '^':
			result.WriteString(`\textasciicircum{}`)
		default:
			result.WriteByte(text[i])
		}
		i++
	}
	return result.String()
}

// ProtectTitle preserves the capitalization and mathematics supplied by the
// source for generated entries, without guessing which words are proper names.
func ProtectTitle(title string) string {
	if title == "" {
		return title
	}
	if title[0] == '{' && braceGroupEnd(title, 0) == len(title) {
		if len(title) > 1 && title[1] == '\\' {
			return "{" + title + "}"
		}
		return title
	}
	if title[0] == '\\' {
		// BibTeX treats a group beginning with a command as a special character.
		// Another group protects the following text from case conversion too.
		return "{{" + title + "}}"
	}
	return "{" + title + "}"
}

func isTeXLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// braceGroupEnd returns the exclusive end of a balanced group, or -1.
func braceGroupEnd(text string, start int) int {
	depth := 0
	for i := start; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) {
			i++
			continue
		}
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}
