package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/clubpay/ronykit/ronyup/knowledge"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const resourceURIPrefix = "knowledge://ronyup/"

func registerResources(srv *mcpsdk.Server, cfg serverConfig) {
	kb := cfg.kb

	srv.AddResourceTemplate(
		&mcpsdk.ResourceTemplate{
			URITemplate: resourceURIPrefix + "{category}/{name}",
			Name:        "RonyKIT Knowledge Base",
			Description: "Architecture hints, package docs, and characteristic " +
				"guidance for RonyKIT service development.",
			MIMEType: "text/markdown",
		},
		resourceTemplateHandler(kb),
	)

	for _, pkg := range kb.Packages {
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + "packages/" + pkg.ShortName,
			Name:        "Package: " + pkg.ShortName,
			Description: truncateText(pkg.Description, 120),
			MIMEType:    "text/markdown",
		}, resourceHandler(kb, "packages", pkg.ShortName))
	}

	for _, hint := range kb.ArchitectureHints {
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + "architecture/" + hint.Slug,
			Name:        "Architecture: " + hint.Slug,
			Description: truncateText(hint.Text, 120),
			MIMEType:    "text/markdown",
		}, resourceHandler(kb, "architecture", hint.Slug))
	}

	for _, ch := range kb.Characteristics {
		name := charResourceName(ch)
		srv.AddResource(&mcpsdk.Resource{
			URI:         resourceURIPrefix + "characteristics/" + name,
			Name:        "Characteristic: " + name,
			Description: truncateText(ch.ServiceHint, 120),
			MIMEType:    "text/markdown",
		}, resourceHandler(kb, "characteristics", name))
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
					MIMEType: "text/markdown",
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
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}

func resolveKnowledge(kb *knowledge.Base, category, name string) string {
	switch category {
	case "packages":
		for _, pkg := range kb.Packages {
			if pkg.ShortName == name {
				return formatPackageResource(pkg)
			}
		}
	case "architecture":
		for _, hint := range kb.ArchitectureHints {
			if hint.Slug == name {
				return hint.Text
			}
		}
	case "characteristics":
		for _, ch := range kb.Characteristics {
			if charResourceName(ch) == name {
				return formatCharacteristicResource(ch)
			}
		}
	}

	return ""
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
	b.WriteString(strings.Join(ch.Keywords, ", "))
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
	if len(ch.Keywords) > 0 {
		return ch.Keywords[0]
	}

	return "unknown"
}

func truncateText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
