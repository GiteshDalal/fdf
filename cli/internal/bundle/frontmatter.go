package bundle

import (
	"fmt"
	"regexp"
	"strings"
)

var keyRe = regexp.MustCompile(`^([A-Za-z0-9_-]+):\s?(.*)$`)
var itemRe = regexp.MustCompile(`^\s*-\s+(.*)$`)

// splitFrontmatter returns the raw block, whether a complete delimited block
// was found, and the remaining body.
func splitFrontmatter(text string) (string, bool, string) {
	if !strings.HasPrefix(text, "---") {
		return "", false, text
	}
	lines := strings.SplitAfter(text, "\n")
	if strings.TrimSpace(strings.TrimSuffix(lines[0], "\n")) != "---" {
		return "", false, text
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimSuffix(lines[i], "\n")) == "---" {
			return strings.Join(lines[1:i], ""), true, strings.Join(lines[i+1:], "")
		}
	}
	return "", false, text
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.Trim(s, `'`)
}

// parseFrontmatter parses the supported YAML subset. Unrecognized non-blank
// lines are a parse error (F1).
func parseFrontmatter(block string) (map[string]any, error) {
	data := map[string]any{}
	lastKey := ""
	for n, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := keyRe.FindStringSubmatch(line); m != nil {
			lastKey = m[1]
			val := strings.TrimSpace(m[2])
			switch {
			case val == "":
				data[lastKey] = []string{}
			case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
				var items []string
				for _, p := range strings.Split(val[1:len(val)-1], ",") {
					if q := unquote(p); q != "" {
						items = append(items, q)
					}
				}
				data[lastKey] = items
			default:
				data[lastKey] = unquote(val)
			}
			continue
		}
		if m := itemRe.FindStringSubmatch(line); m != nil && lastKey != "" {
			if list, ok := data[lastKey].([]string); ok {
				data[lastKey] = append(list, unquote(m[1]))
				continue
			}
		}
		return nil, fmt.Errorf("line %d: not parseable as frontmatter: %q", n+1, trimmed)
	}
	return data, nil
}

// asList normalizes a frontmatter value to a string slice.
func asList(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	default:
		return nil
	}
}
