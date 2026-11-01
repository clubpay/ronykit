package scaffold

import "strings"

func parseSkillFrontmatter(content string) (name, description string, ok bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", "", false
	}

	rest := content[len("---\n"):]

	before, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return "", "", false
	}

	fm := before
	for line := range strings.SplitSeq(fm, "\n") {
		switch {
		case strings.HasPrefix(line, "name:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		case strings.HasPrefix(line, "description:"):
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			if description == ">-" || description == "|" || description == ">" {
				description = "folded"
			}
		}
	}

	if name == "" || description == "" {
		return name, description, false
	}

	return name, description, true
}
