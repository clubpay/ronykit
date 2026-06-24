package intent

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/intent/internal/prompt"
)

const defaultMaxToolIterations = 8

type runtimeConfig struct {
	static            StaticStore
	retriever         Retriever
	pool              Pool
	tools             ToolExecutor
	skills            SkillRegistry
	maxToolIterations int
	logger            *slog.Logger
}

type runtime struct {
	cfg runtimeConfig
	log *slog.Logger
}

func newRuntime(cfg agentConfig) *runtime {
	if cfg.LLM == nil {
		return nil
	}

	maxIter := cfg.maxToolIterations
	if maxIter <= 0 {
		maxIter = defaultMaxToolIterations
	}

	log := cfg.logger
	if log == nil {
		log = slog.Default()
	}

	return &runtime{
		cfg: runtimeConfig{
			static:            cfg.StaticKnowledge,
			retriever:         cfg.Retriever,
			pool:              cfg.LLM,
			tools:             cfg.Tools,
			skills:            cfg.Skills,
			maxToolIterations: maxIter,
			logger:            log,
		},
		log: log,
	}
}

// TurnInput is one user turn inside a session.
type TurnInput struct {
	Session       *Session
	UserMessage   Message
	RetrieveQuery string
}

// TurnResult is the outcome of a completed turn.
type TurnResult struct {
	Response Response
	Messages []Message
}

func (r *runtime) runTurn(ctx context.Context, in TurnInput) (TurnResult, error) {
	if r == nil {
		return TurnResult{}, errs.Wrap(errs.ErrUnsupportedOperation, "agent llm pool is required")
	}

	if in.Session == nil {
		return TurnResult{}, errs.Wrap(errs.ErrSessionNotFound, "session is required")
	}

	r.debug("run turn", "session_id", in.Session.ID)

	messages, err := r.buildContext(ctx, in)
	if err != nil {
		return TurnResult{}, err
	}

	toolDefs, err := r.toolDefinitions(ctx)
	if err != nil {
		return TurnResult{}, err
	}

	skills, err := r.loadSkills(ctx, toolDefs)
	if err != nil {
		return TurnResult{}, err
	}

	newMessages := []Message{in.UserMessage}

	model, err := r.cfg.pool.Select(ctx, in.Session.Selection)
	if err != nil {
		return TurnResult{}, err
	}

	var final Response

	for iteration := range r.cfg.maxToolIterations {
		reqMessages := append([]Message(nil), messages...)
		if iteration == 0 && skills.enabled() {
			reqMessages = append(reqMessages, userMessageWithCatalog(in.UserMessage, skills.cards))
		} else {
			reqMessages = append(reqMessages, newMessages...)
		}

		req := Request{
			Messages: reqMessages,
			Tools:    composeTools(toolDefs, skills),
		}

		r.debug("llm generate", "iteration", iteration, "model_id", model.Model().ID)

		resp, err := model.Generate(ctx, req)
		if err != nil {
			return TurnResult{}, err
		}

		assistant := assistantMessage(resp)
		newMessages = append(newMessages, assistant)

		if len(resp.ToolCalls) == 0 {
			final = resp

			break
		}

		for _, call := range resp.ToolCalls {
			toolMsg := r.handleToolCall(ctx, call, skills)
			newMessages = append(newMessages, toolMsg)
		}

		if iteration == r.cfg.maxToolIterations-1 {
			return TurnResult{}, errs.ErrMaxToolIterations
		}
	}

	err = AppendHistory(ctx, in.Session, newMessages...)
	if err != nil {
		return TurnResult{}, err
	}

	return TurnResult{
		Response: final,
		Messages: newMessages,
	}, nil
}

func (r *runtime) buildContext(ctx context.Context, in TurnInput) ([]Message, error) {
	history, err := LoadHistory(ctx, in.Session)
	if err != nil {
		return nil, err
	}

	var messages []Message

	if r.cfg.static != nil {
		// Skills are advertised on demand via the activate_skill tool, not eagerly
		// injected here; see loadSkills and skillCatalogMessage.
		entries, err := r.cfg.static.List(ctx, Filter{
			Kinds: []Kind{
				KindPrompt,
				KindFact,
			},
		})
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			messages = append(messages, Message{
				Role:  RoleSystem,
				Parts: []Part{TextPart(formatKnowledge(entry))},
			})
		}
	}

	if in.RetrieveQuery != "" && r.cfg.retriever != nil {
		entries, err := r.cfg.retriever.Retrieve(ctx, RetrieveQuery{
			Text:  in.RetrieveQuery,
			Limit: 5,
		})
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			messages = append(messages, Message{
				Role:  RoleSystem,
				Parts: []Part{TextPart(formatKnowledge(entry))},
			})
		}
	}

	messages = append(messages, history...)

	return messages, nil
}

func (r *runtime) toolDefinitions(ctx context.Context) ([]ToolDefinition, error) {
	if r.cfg.tools == nil {
		return nil, nil
	}

	return r.cfg.tools.Definitions(ctx)
}

// handleToolCall routes a single model tool call: skill activation, gated-tool
// rejection, or normal tool execution. It always returns a RoleTool message so
// the conversation stays well-formed even on errors.
func (r *runtime) handleToolCall(ctx context.Context, call ToolCall, skills *skillState) Message {
	if skills.enabled() && call.Name == ActivateSkillTool {
		r.debug("activate skill", "call_id", call.ID, "name", call.Name)

		toolMsg, err := skills.activate(call)
		if err != nil {
			return toolErrorMessage(call, err)
		}

		return toolMsg
	}

	if owner, locked := skills.locked(call.Name); locked {
		r.debug("tool locked", "tool", call.Name, "skill", owner)

		skillDesc := ""
		if skill, ok := skills.skills[owner]; ok {
			skillDesc = skill.Description
		}

		return Message{
			Role: RoleTool,
			Parts: []Part{
				TextPart(prompt.ToolLocked(prompt.ToolLockedData{
					ToolName:          call.Name,
					SkillName:         owner,
					SkillDescription:  skillDesc,
					ActivateSkillTool: ActivateSkillTool,
				})),
			},
			ToolCallID: call.ID,
			ToolName:   call.Name,
		}
	}

	r.debug("execute tool", "tool", call.Name, "call_id", call.ID)

	toolMsg, err := r.executeTool(ctx, call)
	if err != nil {
		return toolErrorMessage(call, err)
	}

	return enrichToolMessage(toolMsg, call)
}

func (r *runtime) executeTool(ctx context.Context, call ToolCall) (Message, error) {
	if r.cfg.tools == nil {
		return Message{}, errs.ToolNotFound(call.Name)
	}

	return r.cfg.tools.Execute(ctx, call)
}

func toolErrorMessage(call ToolCall, err error) Message {
	return Message{
		Role:       RoleTool,
		Parts:      []Part{TextPart(err.Error())},
		ToolCallID: call.ID,
		ToolName:   call.Name,
	}
}

func enrichToolMessage(msg Message, call ToolCall) Message {
	if msg.ToolCallID == "" {
		msg.ToolCallID = call.ID
	}

	if msg.ToolName == "" {
		msg.ToolName = call.Name
	}

	return msg
}

func assistantMessage(resp Response) Message {
	return Message{
		Role: RoleAI,
		Parts: []Part{
			TextPart(resp.Content),
		},
		ToolCalls: resp.ToolCalls,
	}
}

// skillState holds per-turn skill activation and tool-gating state.
//
// Skills advertise only their name and description up front (the catalog).
// Their full Instructions and any skill-scoped Tools enter the turn only after
// the model activates the skill via the activate_skill tool.
type skillState struct {
	skills    map[string]Skill          // name -> full skill
	cards     []SkillCard               // advertised catalog
	defByName map[string]ToolDefinition // every base tool definition, by name
	gated     map[string]string         // gated tool name -> owning skill name
	alwaysOn  []ToolDefinition          // base tools not scoped to any skill
	active    map[string]bool           // activated skill names
}

// loadSkills builds the per-turn skill state. The returned state is always
// non-nil; enabled reports whether any skills are actually available, so the
// turn behaves as before for agents without skills.
func (r *runtime) loadSkills(ctx context.Context, baseDefs []ToolDefinition) (*skillState, error) {
	st := &skillState{
		skills:    make(map[string]Skill),
		defByName: make(map[string]ToolDefinition, len(baseDefs)),
		gated:     make(map[string]string),
		active:    make(map[string]bool),
	}

	for _, def := range baseDefs {
		st.defByName[def.Name] = def
	}

	if r.cfg.skills == nil {
		return st, nil
	}

	cards, err := r.cfg.skills.List(ctx)
	if err != nil {
		return nil, err
	}

	if len(cards) == 0 {
		return st, nil
	}

	st.cards = cards

	for _, card := range cards {
		skill, err := r.cfg.skills.Get(ctx, card.Name)
		if err != nil {
			return nil, err
		}

		st.skills[card.Name] = skill

		for _, tool := range skill.Tools {
			st.gated[tool] = card.Name
		}
	}

	for _, def := range baseDefs {
		if _, gatedOK := st.gated[def.Name]; !gatedOK {
			st.alwaysOn = append(st.alwaysOn, def)
		}
	}

	return st, nil
}

// enabled reports whether any skills are available this turn.
func (st *skillState) enabled() bool {
	return st != nil && len(st.cards) > 0
}

// tools returns the tool definitions visible for the current iteration:
// always-on tools, the activate_skill tool, and the tools of active skills.
func (st *skillState) tools() []ToolDefinition {
	out := make([]ToolDefinition, 0, len(st.alwaysOn)+len(st.gated)+1)
	out = append(out, st.alwaysOn...)
	out = append(out, activateSkillDefinition(st.cards))

	for name := range st.active {
		for _, tool := range st.skills[name].Tools {
			if def, ok := st.defByName[tool]; ok {
				out = append(out, def)
			}
		}
	}

	return out
}

// activate loads a skill's instructions and unlocks its tools for the rest of
// the turn.
func (st *skillState) activate(call ToolCall) (Message, error) {
	name, err := parseSkillName(call.Arguments)
	if err != nil {
		return Message{}, err
	}

	skill, ok := st.skills[name]
	if !ok {
		return Message{}, errs.SkillNotFound(name)
	}

	st.active[name] = true

	return Message{
		Role:       RoleTool,
		Parts:      []Part{TextPart(skill.Instructions)},
		ToolCallID: call.ID,
		ToolName:   call.Name,
	}, nil
}

// locked reports whether a tool call targets a skill-scoped tool whose skill
// has not been activated yet, returning the owning skill name.
func (st *skillState) locked(name string) (string, bool) {
	if st == nil {
		return "", false
	}

	owner, gatedOK := st.gated[name]
	if !gatedOK || st.active[owner] {
		return "", false
	}

	return owner, true
}

// composeTools selects the tool definitions for an iteration: skill-aware when
// skills are configured, otherwise the unmodified base definitions.
func composeTools(base []ToolDefinition, st *skillState) []ToolDefinition {
	if !st.enabled() {
		return base
	}

	return st.tools()
}

func parseSkillName(args string) (string, error) {
	if args == "" {
		return "", errs.Wrap(errs.ErrUnsupportedOperation, "activate_skill requires a name argument")
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "", errs.Wrap(err, "decode activate_skill arguments")
	}

	if payload.Name == "" {
		return "", errs.Wrap(errs.ErrUnsupportedOperation, "activate_skill name is required")
	}

	return payload.Name, nil
}

func formatKnowledge(entry Entry) string {
	return prompt.KnowledgeEntry(prompt.KnowledgeEntryData{
		Name:    entry.Name,
		Content: entry.Content,
		Source:  entry.Source,
	})
}

func (r *runtime) debug(msg string, args ...any) {
	if r.log != nil {
		r.log.Debug(msg, args...)
	}
}
