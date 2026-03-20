package mcp

import (
	"context"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(srv *mcpsdk.Server, cfg serverConfig) {
	for _, p := range cfg.kb.Prompts {
		prompt := p

		args := make([]*mcpsdk.PromptArgument, 0, len(prompt.Arguments))
		for _, a := range prompt.Arguments {
			args = append(args, &mcpsdk.PromptArgument{
				Name:        a.Name,
				Description: a.Description,
				Required:    a.Required,
			})
		}

		srv.AddPrompt(
			&mcpsdk.Prompt{
				Name:        prompt.Name,
				Description: prompt.Description,
				Arguments:   args,
			},
			func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
				rendered := renderPromptTemplate(prompt.Template, req.Params.Arguments)

				return &mcpsdk.GetPromptResult{
					Description: prompt.Description,
					Messages: []*mcpsdk.PromptMessage{
						{
							Role: "user",
							Content: &mcpsdk.TextContent{
								Text: rendered,
							},
						},
					},
				}, nil
			},
		)
	}
}

func renderPromptTemplate(tmpl string, args map[string]string) string {
	result := tmpl

	for key, val := range args {
		result = strings.ReplaceAll(result, "{{"+key+"}}", val)
	}

	result = resolveConditionals(result, args)

	return result
}

// resolveConditionals handles simple {{#if name}}...{{/if}} blocks.
func resolveConditionals(s string, args map[string]string) string {
	for {
		start := strings.Index(s, "{{#if ")
		if start < 0 {
			break
		}

		endTag := strings.Index(s[start:], "}}")
		if endTag < 0 {
			break
		}

		argName := strings.TrimSpace(s[start+len("{{#if ") : start+endTag])

		closeTag := "{{/if}}"

		closeIdx := strings.Index(s[start:], closeTag)
		if closeIdx < 0 {
			break
		}

		body := s[start+endTag+2 : start+closeIdx]

		val := strings.TrimSpace(args[argName])
		if val != "" {
			s = s[:start] + strings.TrimSpace(body) + s[start+closeIdx+len(closeTag):]
		} else {
			s = s[:start] + s[start+closeIdx+len(closeTag):]
		}
	}

	return strings.TrimSpace(s)
}
