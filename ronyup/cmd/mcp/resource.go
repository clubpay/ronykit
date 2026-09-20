package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	resourceURIPrefix       = "knowledge://ronyup/"
	markdownMIMEType        = "text/markdown"
	categoryPackages        = "packages"
	categoryArchitecture    = "architecture"
	categoryCharacteristics = "characteristics"
	categoryTools           = "tools"
)

var resourceTemplateURI = resourceURIPrefix + "{category}/{name}"

func registerResources(srv *mcpsdk.Server, cfg ServerConfig) {
	kb := cfg.kb

	srv.AddResourceTemplate(
		&mcpsdk.ResourceTemplate{
			URITemplate: resourceTemplateURI,
			Name:        "RonyKIT Knowledge Base",
			Description: "Architecture hints, package docs, and characteristic " +
				"guidance for RonyKIT service development.",
			MIMEType: markdownMIMEType,
		},
		resourceTemplateHandler(kb),
	)

	for _, pkg := range kb.Packages {
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + categoryPackages + "/" + pkg.ShortName,
			Name:        "Package: " + pkg.ShortName,
			Description: resourceDescription(pkg.Description),
			MIMEType:    markdownMIMEType,
		}, resourceHandler(kb, categoryPackages, pkg.ShortName))
	}

	for _, hint := range kb.ArchitectureHints {
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + categoryArchitecture + "/" + hint.Slug,
			Name:        "Architecture: " + hint.Slug,
			Description: resourceDescription(hint.Text),
			MIMEType:    markdownMIMEType,
		}, resourceHandler(kb, categoryArchitecture, hint.Slug))
	}

	for _, ch := range kb.Characteristics {
		name := charResourceName(ch)
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + categoryCharacteristics + "/" + name,
			Name:        "Characteristic: " + name,
			Description: resourceDescription(ch.ServiceHint),
			MIMEType:    markdownMIMEType,
		}, resourceHandler(kb, categoryCharacteristics, name))
	}

	for _, tool := range kb.Tools {
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + categoryTools + "/" + tool.Name,
			Name:        "Tool: " + tool.Name,
			Description: resourceDescription(tool.Description),
			MIMEType:    markdownMIMEType,
		}, resourceHandler(kb, categoryTools, tool.Name))
	}
}

func resourceTemplateHandler(kb *knowledge.Base) mcpsdk.ResourceHandler {
	return func(_ context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		uri := strings.TrimPrefix(req.Params.URI, resourceURIPrefix)

		parts := strings.SplitN(uri, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid knowledge URI: %s", req.Params.URI)
		}

		content := resolveKnowledge(kb, parts[0], parts[1])
		if content == "" {
			return nil, fmt.Errorf("knowledge not found: %s/%s", parts[0], parts[1])
		}

		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: markdownMIMEType,
					Text:     content,
				},
			},
		}, nil
	}
}

func resourceHandler(kb *knowledge.Base, category, name string) mcpsdk.ResourceHandler {
	return func(_ context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		content := resolveKnowledge(kb, category, name)
		if content == "" {
			return nil, fmt.Errorf("knowledge not found: %s/%s", category, name)
		}

		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: markdownMIMEType,
					Text:     content,
				},
			},
		}, nil
	}
}

func resolveKnowledge(kb *knowledge.Base, category, name string) string {
	switch category {
	case categoryPackages:
		for _, pkg := range kb.Packages {
			if pkg.ShortName == name {
				return formatPackageResource(pkg)
			}
		}
	case categoryArchitecture:
		for _, hint := range kb.ArchitectureHints {
			if hint.Slug == name {
				return hint.Text
			}
		}
	case categoryCharacteristics:
		for _, ch := range kb.Characteristics {
			if characteristicMatches(ch, name) {
				return formatCharacteristicResource(ch)
			}
		}
	case categoryTools:
		for _, tool := range kb.Tools {
			if tool.Name == name {
				return formatToolResource(tool)
			}
		}
	}

	return ""
}

func formatToolResource(tool knowledge.ToolDoc) string {
	var b strings.Builder
	b.WriteString("# Tool: ")
	b.WriteString(tool.Name)
	b.WriteString("\n\n")
	b.WriteString(tool.Description)

	if tool.ExtendedGuidance != "" {
		b.WriteString("\n\n## Extended Guidance\n\n")
		b.WriteString(tool.ExtendedGuidance)
	}

	b.WriteByte('\n')

	return b.String()
}

func formatPackageResource(pkg knowledge.PackageDoc) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(pkg.ShortName)
	b.WriteString("\n\n**Import:** `")
	b.WriteString(pkg.ImportPath)
	b.WriteString("`\n\n")
	b.WriteString(pkg.Description)

	if pkg.UsageHint != "" {
		b.WriteString("\n\n## Usage Hint\n\n")
		b.WriteString(pkg.UsageHint)
	}

	b.WriteByte('\n')

	return b.String()
}

func formatCharacteristicResource(ch knowledge.CharacteristicDoc) string {
	var b strings.Builder
	b.WriteString("# Characteristic: ")
	b.WriteString(charResourceName(ch))
	b.WriteString("\n\n")
	b.WriteString(ch.ServiceHint)

	if ch.FileHint != "" {
		b.WriteString("\n\n## File-Level Hint\n\n")
		b.WriteString(ch.FileHint)
	}

	b.WriteByte('\n')

	return b.String()
}

func charResourceName(ch knowledge.CharacteristicDoc) string {
	if ch.Name != "" {
		return ch.Name
	}

	if ch.Slug != "" {
		return ch.Slug
	}

	return "unknown"
}

func characteristicMatches(ch knowledge.CharacteristicDoc, name string) bool {
	if charResourceName(ch) == name {
		return true
	}

	return slices.Contains(ch.Keywords, name)
}

func characteristicNames(ch knowledge.CharacteristicDoc) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, 1+len(ch.Keywords))

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}

		if _, ok := seen[s]; ok {
			return
		}

		seen[s] = struct{}{}
		names = append(names, s)
	}

	add(charResourceName(ch))

	for _, kw := range ch.Keywords {
		add(kw)
	}

	return names
}

func completionHandler(
	kb *knowledge.Base,
) func(context.Context, *mcpsdk.CompleteRequest) (*mcpsdk.CompleteResult, error) {
	return func(_ context.Context, req *mcpsdk.CompleteRequest) (*mcpsdk.CompleteResult, error) {
		if req == nil || req.Params == nil || req.Params.Ref == nil {
			return emptyCompletion(), nil
		}

		refType := strings.ToLower(strings.TrimSpace(req.Params.Ref.Type))
		refURI := strings.TrimSpace(req.Params.Ref.URI)

		switch refType {
		case "ref/resource":
			if refURI != resourceTemplateURI {
				return emptyCompletion(), nil
			}

			return completeResource(kb, req), nil
		default:
			return emptyCompletion(), nil
		}
	}
}

func completeResource(
	kb *knowledge.Base, req *mcpsdk.CompleteRequest,
) *mcpsdk.CompleteResult {
	argName := strings.ToLower(strings.TrimSpace(req.Params.Argument.Name))
	argValue := strings.TrimSpace(req.Params.Argument.Value)

	var candidates []string

	switch argName {
	case "category":
		candidates = filterPrefix(knowledgeCategories(), argValue)
	case "name":
		category := ""
		if req.Params.Context != nil {
			category = req.Params.Context.Arguments["category"]
		}

		candidates = filterPrefix(namesForCategory(kb, category), argValue)
	}

	return buildCompletionResult(candidates)
}

func knowledgeCategories() []string {
	return []string{categoryPackages, categoryArchitecture, categoryCharacteristics, categoryTools}
}

const maxCompletionValues = 64

func buildCompletionResult(candidates []string) *mcpsdk.CompleteResult {
	if candidates == nil {
		candidates = []string{}
	}

	total := len(candidates)
	hasMore := total > maxCompletionValues

	if hasMore {
		candidates = candidates[:maxCompletionValues]
	}

	return &mcpsdk.CompleteResult{
		Completion: mcpsdk.CompletionResultDetails{
			Values:  candidates,
			Total:   total,
			HasMore: hasMore,
		},
	}
}

func emptyCompletion() *mcpsdk.CompleteResult {
	return &mcpsdk.CompleteResult{
		Completion: mcpsdk.CompletionResultDetails{
			Values: []string{},
		},
	}
}

func namesForCategory(kb *knowledge.Base, category string) []string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case categoryPackages:
		names := make([]string, 0, len(kb.Packages))
		for _, pkg := range kb.Packages {
			names = append(names, pkg.ShortName)
		}

		return names
	case categoryArchitecture:
		names := make([]string, 0, len(kb.ArchitectureHints))
		for _, hint := range kb.ArchitectureHints {
			names = append(names, hint.Slug)
		}

		return names
	case categoryCharacteristics:
		names := make([]string, 0, len(kb.Characteristics))
		for _, ch := range kb.Characteristics {
			names = append(names, characteristicNames(ch)...)
		}

		return names
	case categoryTools:
		names := make([]string, 0, len(kb.Tools))
		for _, tool := range kb.Tools {
			names = append(names, tool.Name)
		}

		return names
	default:
		var all []string

		for _, pkg := range kb.Packages {
			all = append(all, pkg.ShortName)
		}

		for _, hint := range kb.ArchitectureHints {
			all = append(all, hint.Slug)
		}

		for _, ch := range kb.Characteristics {
			all = append(all, characteristicNames(ch)...)
		}

		for _, tool := range kb.Tools {
			all = append(all, tool.Name)
		}

		return all
	}
}

func filterPrefix(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}

	lowerPrefix := strings.ToLower(prefix)

	var result []string

	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), lowerPrefix) {
			result = append(result, item)
		}
	}

	return result
}

func resourceDescription(s string) string {
	const maxLen = 120

	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
