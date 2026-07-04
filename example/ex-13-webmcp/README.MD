# ex-13-webmcp — agent-friendly task board

Minimal **Vite + React** demo showing how to expose a web app as
[WebMCP](https://webmachinelearning.github.io/webmcp/) tools using
[`@mcp-b/global`](https://docs.mcp-b.ai) and
[`@mcp-b/react-webmcp`](https://docs.mcp-b.ai/packages/react-webmcp/reference).

The human UI and agent tools share the same task logic in `src/lib/tasks.ts`.

## Tools exposed

| Tool | R/W | Purpose |
|------|-----|---------|
| `tasks_list` | read | List tasks; optional `status` filter |
| `tasks_add` | write | Add a task by title |
| `tasks_toggle` | write | Toggle done/open by task `id` |
| `tasks_remove` | write | Delete a task by `id` |
| `tasks_stats` | read | Counts of open/done/total tasks |

## Run locally

```bash
cd example/ex-13-webmcp
npm install   # or pnpm install (Node 22+)
npm run dev
```

Open http://localhost:5173 — use the UI or test tools in the browser console:

```javascript
// After page load (with @mcp-b/global polyfill active)
const tools = navigator.modelContextTesting?.listTools?.() ?? [];
console.log(tools.map((t) => t.name));

const raw = await navigator.modelContextTesting.executeTool(
  'tasks_add',
  JSON.stringify({ title: 'Buy milk' }),
);
console.log(JSON.parse(raw));
```

Native WebMCP (Chrome Canary + `chrome://flags/#enable-webmcp-testing`) uses the
same tool names; the polyfill steps aside when native support exists.

## Test

```bash
npm test
npm run build
```

## Related

- Skill: `ronyup/internal/skeleton/skills/webmcp/`
- W3C spec: https://webmachinelearning.github.io/webmcp/
- MCP-B docs: https://docs.mcp-b.ai

This example is instructional — not for production.
