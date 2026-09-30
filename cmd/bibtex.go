package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nickng/bibtex"
)

// formatBibTeX aligns fields while preserving their contents verbatim.
func formatBibTeX(entry *bibtex.BibEntry) string {
	keys := make([]string, 0, len(entry.Fields))
	width := 0
	for key := range entry.Fields {
		keys = append(keys, key)
		if len(key) > width {
			width = len(key)
		}
	}
	priority := map[string]int{"title": -3, "author": -2, "url": -1}
	sort.Slice(keys, func(i, j int) bool {
		pi, pj := priority[keys[i]], priority[keys[j]]
		return pi < pj || (pi == pj && keys[i] < keys[j])
	})

	var output strings.Builder
	fmt.Fprintf(&output, "@%s{%s,\n", entry.Type, entry.CiteName)
	for _, key := range keys {
		value := entry.Fields[key].String()
		if _, err := strconv.Atoi(value); err != nil {
			// BibTeX delimiters preserve LaTeX commands and newlines; Go's %q escapes them.
			if strings.ContainsAny(value, "\"{}") {
				value = "{" + value + "}"
			} else {
				value = "\"" + value + "\""
			}
		}
		fmt.Fprintf(&output, "    %-*s = %s,\n", width, key, value)
	}
	output.WriteString("}\n")
	return output.String()
}
