package gosdk

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"

	"github.com/clubpay/ronykit/intent"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Factory creates MCP client connections using the official Go SDK.
type Factory struct {
	client *sdk.Client
}

// NewFactory returns a factory with a default MCP client implementation.
func NewFactory(name string) *Factory {
	return &Factory{
		client: sdk.NewClient(&sdk.Implementation{Name: name}, nil),
	}
}

func (f *Factory) NewServer(ctx context.Context, cfg intent.MCPServerConfig) (intent.MCPServer, error) {
	if f == nil || f.client == nil {
		return nil, fmt.Errorf("mcp factory is nil")
	}

	s := &Server{name: cfg.Name, cfg: cfg, client: f.client}

	err := s.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

var _ intent.MCPClientFactory = (*Factory)(nil)

// Server adapts sdk.ClientSession to intent.MCPServer.
type Server struct {
	name    string
	cfg     intent.MCPServerConfig
	client  *sdk.Client
	session *sdk.ClientSession
}

func (s *Server) Name() string { return s.name }

func (s *Server) Connect(ctx context.Context) error {
	if s.session != nil {
		return nil
	}

	transport, err := transportForConfig(ctx, s.cfg)
	if err != nil {
		return err
	}

	session, err := s.client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}

	s.session = session

	return nil
}

func (s *Server) Close() error {
	if s.session == nil {
		return nil
	}

	err := s.session.Close()
	s.session = nil

	return err
}

func (s *Server) ListTools(ctx context.Context) ([]intent.MCPTool, error) {
	if err := s.requireSession(); err != nil {
		return nil, err
	}

	res, err := s.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	out := make([]intent.MCPTool, 0, len(res.Tools))
	for _, tool := range res.Tools {
		out = append(out, intent.MCPTool{
			Name:         tool.Name,
			Title:        tool.Title,
			Description:  tool.Description,
			InputSchema:  marshalSchema(tool.InputSchema),
			OutputSchema: marshalSchema(tool.OutputSchema),
		})
	}

	return out, nil
}

func (s *Server) CallTool(ctx context.Context, name string, args json.RawMessage) (intent.MCPToolResult, error) {
	if err := s.requireSession(); err != nil {
		return intent.MCPToolResult{}, err
	}

	var arguments any
	if len(args) > 0 {
		err := json.Unmarshal(args, &arguments)
		if err != nil {
			return intent.MCPToolResult{}, err
		}
	}

	res, err := s.session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return intent.MCPToolResult{}, err
	}

	return fromCallToolResult(res), nil
}

func (s *Server) ListResources(ctx context.Context) ([]intent.MCPResourceSummary, error) {
	if err := s.requireSession(); err != nil {
		return nil, err
	}

	res, err := s.session.ListResources(ctx, nil)
	if err != nil {
		return nil, err
	}

	out := make([]intent.MCPResourceSummary, 0, len(res.Resources))
	for _, item := range res.Resources {
		out = append(out, intent.MCPResourceSummary{
			URI:         item.URI,
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
			MIMEType:    item.MIMEType,
		})
	}

	return out, nil
}

func (s *Server) ReadResource(ctx context.Context, uri string) (intent.MCPResourceContent, error) {
	if err := s.requireSession(); err != nil {
		return intent.MCPResourceContent{}, err
	}

	res, err := s.session.ReadResource(ctx, &sdk.ReadResourceParams{URI: uri})
	if err != nil {
		return intent.MCPResourceContent{}, err
	}

	content := intent.MCPResourceContent{URI: uri}

	if len(res.Contents) > 0 {
		first := res.Contents[0]
		content.MIMEType = first.MIMEType
		content.Text = first.Text
		content.Binary = first.Blob
	}

	return content, nil
}

func (s *Server) ListPrompts(ctx context.Context) ([]intent.MCPPromptSummary, error) {
	if err := s.requireSession(); err != nil {
		return nil, err
	}

	res, err := s.session.ListPrompts(ctx, nil)
	if err != nil {
		return nil, err
	}

	out := make([]intent.MCPPromptSummary, 0, len(res.Prompts))
	for _, item := range res.Prompts {
		out = append(out, intent.MCPPromptSummary{
			Name:        item.Name,
			Title:       item.Title,
			Description: item.Description,
		})
	}

	return out, nil
}

func (s *Server) GetPrompt(ctx context.Context, name string, args map[string]string) (intent.MCPPromptResult, error) {
	if err := s.requireSession(); err != nil {
		return intent.MCPPromptResult{}, err
	}

	params := &sdk.GetPromptParams{Name: name}

	if len(args) > 0 {
		paramsArguments := make(map[string]string, len(args))
		maps.Copy(paramsArguments, args)

		params.Arguments = paramsArguments
	}

	res, err := s.session.GetPrompt(ctx, params)
	if err != nil {
		return intent.MCPPromptResult{}, err
	}

	out := intent.MCPPromptResult{Description: res.Description}
	for _, msg := range res.Messages {
		out.Messages = append(out.Messages, promptMessageFromSDK(msg))
	}

	return out, nil
}

func (s *Server) requireSession() error {
	if s.session == nil {
		return fmt.Errorf("mcp server %q is not connected", s.name)
	}

	return nil
}

var _ intent.MCPServer = (*Server)(nil)

func transportForConfig(ctx context.Context, cfg intent.MCPServerConfig) (sdk.Transport, error) {
	switch cfg.Transport {
	case intent.MCPTransportStreamableHTTP, "":
		if cfg.URL == "" {
			return nil, fmt.Errorf("streamable-http transport requires url")
		}

		return &sdk.StreamableClientTransport{Endpoint: cfg.URL}, nil
	case intent.MCPTransportSSE:
		if cfg.URL == "" {
			return nil, fmt.Errorf("sse transport requires url")
		}

		return &sdk.SSEClientTransport{Endpoint: cfg.URL}, nil
	case intent.MCPTransportStdio:
		if len(cfg.Command) == 0 {
			return nil, fmt.Errorf("stdio transport requires command")
		}

		cmd := exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)

		return &sdk.CommandTransport{Command: cmd}, nil
	default:
		return nil, fmt.Errorf("unsupported mcp transport %q", cfg.Transport)
	}
}

func marshalSchema(v any) json.RawMessage {
	if v == nil {
		return nil
	}

	switch schema := v.(type) {
	case json.RawMessage:
		return schema
	default:
		bb, err := json.Marshal(v)
		if err != nil {
			return nil
		}

		return bb
	}
}

func fromCallToolResult(res *sdk.CallToolResult) intent.MCPToolResult {
	if res == nil {
		return intent.MCPToolResult{}
	}

	out := intent.MCPToolResult{
		IsError: res.IsError,
	}
	if res.StructuredContent != nil {
		bb, err := json.Marshal(res.StructuredContent)
		if err == nil {
			out.StructuredContent = bb
		}
	}

	for _, content := range res.Content {
		switch c := content.(type) {
		case *sdk.TextContent:
			out.Content = append(out.Content, intent.MCPContentBlock{Text: c.Text})
		case *sdk.ImageContent:
			out.Content = append(out.Content, intent.MCPContentBlock{
				MIMEType: c.MIMEType,
				Binary:   c.Data,
			})
		case *sdk.ResourceLink:
			out.Content = append(out.Content, intent.MCPContentBlock{URI: c.URI, MIMEType: c.MIMEType})
		}
	}

	return out
}

func promptMessageFromSDK(msg *sdk.PromptMessage) intent.MCPPromptMessage {
	if msg == nil {
		return intent.MCPPromptMessage{}
	}

	out := intent.MCPPromptMessage{Role: string(msg.Role)}
	switch content := msg.Content.(type) {
	case *sdk.TextContent:
		out.Text = content.Text
	case *sdk.ImageContent:
		out.MIMEType = content.MIMEType
		out.Binary = content.Data
	}

	return out
}
