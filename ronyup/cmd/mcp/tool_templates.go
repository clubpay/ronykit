package mcp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type templateFile struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type listTemplatesOutput struct {
	Templates []templateFile `json:"templates"`
}

type readTemplateInput struct {
	Path string `json:"path" jsonschema:"required,description:Template path listed by list_templates"`
}

type readTemplateOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// Tool registrars
// ---------------------------------------------------------------------------

func registerListTemplates(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolListTemplates,
		Description: cfg.kb.ToolDescription(toolListTemplates),
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, _ struct{},
	) (*mcpsdk.CallToolResult, listTemplatesOutput, error) {
		files, err := collectSkeletonFiles(cfg.skeletonFS)
		if err != nil {
			return nil, listTemplatesOutput{}, err
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf(msgFoundTemplates, len(files)),
				},
			},
		}, listTemplatesOutput{Templates: files}, nil
	})
}

func registerReadTemplate(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolReadTemplate,
		Description: cfg.kb.ToolDescription(toolReadTemplate),
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, in readTemplateInput,
	) (*mcpsdk.CallToolResult, readTemplateOutput, error) {
		normalizedPath, err := normalizeTemplatePath(in.Path)
		if err != nil {
			return nil, readTemplateOutput{}, err
		}

		data, err := fs.ReadFile(cfg.skeletonFS, normalizedPath)
		if err != nil {
			return nil, readTemplateOutput{}, err
		}

		out := readTemplateOutput{
			Path:    normalizedPath,
			Content: string(data),
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf(msgLoadedTemplate, normalizedPath, len(data)),
				},
			},
		}, out, nil
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func collectSkeletonFiles(skeletonFS fs.FS) ([]templateFile, error) {
	var files []templateFile

	err := fs.WalkDir(skeletonFS, "skeleton", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		files = append(files, templateFile{
			Path: filePath,
			Kind: fileKind(filePath),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func normalizeTemplatePath(filePath string) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", errors.New(errTemplatePathRequired)
	}

	cleanPath := path.Clean(strings.TrimPrefix(filePath, "/"))
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") || cleanPath == ".." {
		return "", fmt.Errorf(errInvalidTemplatePath, filePath)
	}

	if !strings.HasPrefix(cleanPath, "skeleton/") {
		cleanPath = path.Join("skeleton", cleanPath)
	}

	return cleanPath, nil
}

func fileKind(filePath string) string {
	if strings.HasSuffix(filePath, ".gotmpl") ||
		strings.HasSuffix(filePath, ".yamltmpl") ||
		strings.HasSuffix(filePath, "tmpl") {
		return "template"
	}

	return "file"
}
