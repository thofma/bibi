package bibfile

import (
	"fmt"
	"strings"

	"github.com/thofma/bibi/lib/bibliography"
)

func resolveValue(value bibValue, macros map[string]macroValue) (string, error) {
	var text strings.Builder
	for _, part := range value {
		if !part.macro {
			text.WriteString(part.text)
			continue
		}
		macro, found := macros[strings.ToLower(part.text)]
		if !found {
			return "", fmt.Errorf("undefined string macro %q", part.text)
		}
		if macro.err != nil {
			return "", fmt.Errorf("string macro %q: %w", part.text, macro.err)
		}
		text.WriteString(macro.text)
	}
	return text.String(), nil
}

func entryIdentifiers(fields map[string]bibValue, macros map[string]macroValue) (map[string]string, error) {
	ids := make(map[string]string)
	for _, field := range []string{"doi", "mrnumber", "zbmath", "zbl"} {
		parts, exists := fields[field]
		if !exists {
			continue
		}
		value, err := resolveValue(parts, macros)
		if err != nil {
			return nil, fmt.Errorf("cannot evaluate %s: %w", field, err)
		}
		value = ungroup(value)
		if value == "" {
			continue
		}
		switch field {
		case "doi":
			value, exists = bibliography.DOIQuery(value)
		case "mrnumber":
			if len(value) >= 2 && strings.EqualFold(value[:2], "mr") {
				value = value[2:]
			}
			exists = decimal(value)
			value = number(value)
		case "zbmath":
			exists = decimal(value)
			value = number(value)
		case "zbl":
			if len(value) >= 3 && strings.EqualFold(value[:3], "zbl") {
				value = strings.TrimSpace(value[3:])
			}
			parts := strings.Split(value, ".")
			exists = len(parts) == 2 && decimal(parts[0]) && decimal(parts[1])
			if exists {
				value = number(parts[0]) + "." + number(parts[1])
			}
		}
		if !exists {
			return nil, fmt.Errorf("invalid %s identifier %q", field, value)
		}
		ids[field] = value
	}
	// arXiv versions remain distinct. Other eprint services do not share this ID.
	if eprint, exists := fields["eprint"]; exists {
		for _, kind := range []string{"archiveprefix", "eprinttype"} {
			service, err := resolveValue(fields[kind], macros)
			if err != nil {
				return nil, fmt.Errorf("cannot evaluate %s: %w", kind, err)
			}
			if !strings.EqualFold(ungroup(service), "arxiv") {
				continue
			}
			value, err := resolveValue(eprint, macros)
			if err != nil {
				return nil, fmt.Errorf("cannot evaluate eprint: %w", err)
			}
			value = strings.ToLower(ungroup(value))
			value = strings.TrimPrefix(value, "arxiv:")
			if value = strings.TrimSpace(value); value != "" {
				ids["arxiv"] = value
			}
			break
		}
	}
	return ids, nil
}

func ungroup(value string) string {
	value = strings.TrimSpace(value)
	for strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		// Remove only a brace group enclosing the entire value.
		depth, whole := 0, true
		for i, char := range value {
			if char == '{' {
				depth++
			} else if char == '}' {
				depth--
			}
			if depth == 0 && i < len(value)-1 {
				whole = false
				break
			}
		}
		if !whole {
			break
		}
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	return value
}

func number(value string) string {
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return "0"
	}
	return value
}

func duplicateKeys(index fileIndex, incoming indexedEntry) ([]string, error) {
	var keys []string
	for _, existing := range index.entries {
		match, conflict := false, ""
		for _, field := range []string{"doi", "mrnumber", "zbmath", "zbl", "arxiv"} {
			left, right := existing.ids[field], incoming.ids[field]
			if left == "" || right == "" {
				continue
			}
			if left == right {
				match = true
			} else {
				conflict = fmt.Sprintf("%s differs: existing=%q incoming=%q", field, left, right)
			}
		}
		if match && conflict != "" {
			return nil, fmt.Errorf("conflicting identifiers with entry %q: %s", existing.key, conflict)
		}
		if !match && strings.EqualFold(existing.key, incoming.key) {
			return nil, fmt.Errorf("citation key %q is already used without a verified identity match; choose another key with --key", existing.key)
		}
		if match {
			keys = append(keys, existing.key)
		}
	}
	return keys, nil
}
