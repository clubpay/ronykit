package intent

import "context"

// Adapter notes for github.com/tmc/langchaingo/llms:
//
//   - LLM.Generate      -> llms.Model.GenerateContent
//   - LLM.Stream        -> GenerateContent with llms.WithStreamingFunc
//   - Message/Part      -> llms.MessageContent + llms.ContentPart
//   - Role              -> llms.ChatMessageType
//   - ToolDefinition    -> llms.FunctionDefinition (WithFunctions / tools API)
//   - ToolCall          -> llms.ToolCall + llms.FunctionCall
//   - GenerateOptions   -> llms.CallOption helpers (WithModel, WithTemperature, ...)
//   - Response.Choices  -> llms.ContentResponse.Choices
//
// Tool-result turns use RoleTool with ToolCallID/ToolName or Part.ToolCallID.

// Role identifies who produced a message.
// Values align with github.com/tmc/langchaingo/llms.ChatMessageType.
type Role string

const (
	RoleSystem  Role = "system"
	RoleHuman   Role = "human"
	RoleAI      Role = "ai"
	RoleTool    Role = "tool"
	RoleGeneric Role = "generic"
)

// Part is one segment of a message. Text-only agents use TextPart.
// Adapters map richer SDK parts (e.g. langchaingo ContentPart) into this shape.
type Part struct {
	Text       string `json:"text,omitempty"`
	MIMEType   string `json:"mimeType,omitempty"`
	Binary     []byte `json:"binary,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
}

// TextPart builds a single-text message part.
func TextPart(text string) Part {
	return Part{Text: text}
}

// Message is a chat turn. Multipart and tool-result messages are supported.
type Message struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts,omitempty"`

	// ToolCalls is set on assistant messages when the model requests tool.
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`

	// ToolCallID and ToolName are set on tool-result messages.
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
}

// ToolDefinition describes a tool the model may call.
// Parameters hold JSON Schema (object). Adapters map to langchaingo FunctionDefinition.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  any
	Strict      bool
}

// GenerateOptions controls generation behavior.
// Adapters map these to langchaingo llms.CallOption values.
type GenerateOptions struct {
	Model            string
	Temperature      *float64
	MaxTokens        *int
	StopWords        []string
	JSONMode         bool
	CandidateCount   int
	Streaming        bool
	ProviderSpecific any // opaque passthrough for adapter-specific options
}

// Request is an LLM completion request.
type Request struct {
	Messages []Message
	Tools    []ToolDefinition
	Options  GenerateOptions
}

// ToolCall is a model-initiated tool invocation.
// Adapters map to langchaingo llms.ToolCall.
type ToolCall struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Response is a non-streaming LLM completion result.
// When the backend returns multiple candidates, implementations should populate
// Choices; the first choice mirrors the top-level fields for convenience.
type Response struct {
	Content          string
	ToolCalls        []ToolCall
	FinishReason     string
	ReasoningContent string
	GenerationInfo   map[string]any
	Choices          []Choice
}

// Choice is one completion candidate.
type Choice struct {
	Content          string
	ToolCalls        []ToolCall
	FinishReason     string
	ReasoningContent string
	GenerationInfo   map[string]any
}

// Chunk is one streaming update.
type Chunk struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	FinishReason     string
	Done             bool
}

// Stream delivers incremental LLM output.
// langchaingo adapters typically buffer WithStreamingFunc callbacks into Recv.
type Stream interface {
	Recv(ctx context.Context) (Chunk, error)
	Close() error
}

// Model describes a connected LLM endpoint.
type Model struct {
	ID       string
	Name     string
	Priority int
}

// LLM is a single language-model backend.
// Adapters wrap github.com/tmc/langchaingo/llms.Model.
type LLM interface {
	Model() Model
	Generate(ctx context.Context, req Request) (Response, error)
	Stream(ctx context.Context, req Request) (Stream, error)
}

// Pool is the set of LLMs available to an agent together with a selection strategy.
type Pool interface {
	Models() []Model
	Select(ctx context.Context, sel Selection) (LLM, error)
}
