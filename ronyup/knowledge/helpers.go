package knowledge

import "strings"

const defaultCharacteristicHint = "Implement as app-layer use-case behavior and expose through explicit contracts."

// ToolDescription returns the description for the named tool, or an empty
// string if the tool is not found in the knowledge base.
func (b *Base) ToolDescription(name string) string {
	if doc, ok := b.Tools[name]; ok {
		return doc.Description
	}

	return ""
}

// ArchitectureHintTexts returns the text of every architecture hint.
func (b *Base) ArchitectureHintTexts() []string {
	out := make([]string, 0, len(b.ArchitectureHints))
	for _, h := range b.ArchitectureHints {
		out = append(out, h.Text)
	}

	return out
}

// CharacteristicHints maps user-provided characteristic strings to the
// matching service-level hint from the knowledge base.
func (b *Base) CharacteristicHints(characteristics []string) map[string]string {
	hints := make(map[string]string, len(characteristics))

	for _, raw := range characteristics {
		norm := normalizeCharacteristic(raw)
		if norm == "" {
			continue
		}

		matched := false

		for _, doc := range b.Characteristics {
			if matchesKeywords(norm, doc.Keywords) {
				hints[raw] = doc.ServiceHint
				matched = true

				break
			}
		}

		if !matched {
			hints[raw] = defaultCharacteristicHint
		}
	}

	return hints
}

// FileHints returns file-level hints applicable to the given file path
// based on the user-provided characteristics.
func (b *Base) FileHints(filePath string, characteristics []string) []string {
	lowerPath := strings.ToLower(filePath)
	seen := map[string]struct{}{}

	var hints []string

	for _, raw := range characteristics {
		norm := normalizeCharacteristic(raw)
		if norm == "" {
			continue
		}

		for _, doc := range b.Characteristics {
			if !matchesKeywords(norm, doc.Keywords) {
				continue
			}

			if doc.FileHint == "" {
				continue
			}

			if len(doc.AppliesToFiles) == 0 {
				if _, dup := seen[doc.FileHint]; !dup {
					seen[doc.FileHint] = struct{}{}
					hints = append(hints, doc.FileHint)
				}

				continue
			}

			for _, frag := range doc.AppliesToFiles {
				if strings.Contains(lowerPath, frag) ||
					(frag == "migration" && strings.HasSuffix(lowerPath, "migration.go")) {
					if _, dup := seen[doc.FileHint]; !dup {
						seen[doc.FileHint] = struct{}{}
						hints = append(hints, doc.FileHint)
					}

					break
				}
			}
		}
	}

	return hints
}

func normalizeCharacteristic(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "_", " ")
	v = strings.ReplaceAll(v, "-", " ")

	return v
}

func matchesKeywords(normalized string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(normalized, kw) {
			return true
		}
	}

	return false
}
