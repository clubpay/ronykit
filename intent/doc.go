// Package intent provides a goal-driven agent framework built on RonyKit.
//
// An Agent wraps a rony.Server and composes knowledge, LLM pools, memory,
// MCP servers, and rony endpoints. Agents expose HTTP/RPC handlers through
// ServiceDescriptor and EndpointMount without passing *rony.Server to user code.
//
// # Core types
//
// All agent building blocks live in this package:
//
//   - Embedder — vector embedding for memory and RAG backends
//   - Knowledge, StaticStore, Retriever, Indexer — static catalog and RAG
//   - LLM, Pool, Selector — language model backends and selection
//   - Memory, SessionMemory — session-scoped conversation state
//   - MCPServer, MCPRegistry — external MCP tool and context sources
//   - LocalTool, ToolRegistry — local and MCP-backed executable tools
//   - Skill, SkillRegistry — on-demand capabilities loaded via activate_skill
//   - Agent — entry point for turns, sessions, and server lifecycle
//   - ServiceDescriptor, EndpointMount, ServiceRegistry — rony endpoints
//   - TaskExecutor, TaskHandle, TaskState — durable task lifecycle
//
// Subpackages:
//
//   - intent/errs — shared error sentinels
package intent
