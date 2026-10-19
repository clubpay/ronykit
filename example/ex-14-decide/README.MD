# ex-14-decide: ticket triage service on rony + x/decide

A support-ticket triage API served by `rony` and powered by `x/decide`. It demonstrates:

- **Self-contained enum & rubric types** (`Department.Criteria()`, `Urgency.Levels()`, `Urgency.MinLevel()`)
- **Self-describing schema struct** (`TriageDecision.Describe(b *decide.Builder[TriageDecision])`)
- **Template-based state generation** (`decide.MustTemplate[TicketInput](...)`) to format structured ticket fields into controlled state text before hitting the model
- **Pluggable backends**: local demo resolver out of the box, or **Laya**, **TypeSafe Jev**, or **OpenAI-compatible LLM** via environment variables

```sh
# Run with the built-in local demo resolver (no API key needed):
go run .   # http://127.0.0.1:8080, API docs at /docs

# Or switch to Laya (laya-serve or Laya Studio):
DECIDE_PROVIDER=laya DECIDE_LAYA_ENDPOINT=http://127.0.0.1:8000/v1/systemone go run .

# Or switch to TypeSafe Jev:
DECIDE_PROVIDER=jev DECIDE_JEV_API_KEY=your_key go run .

# Or switch to an OpenAI-compatible LLM:
DECIDE_PROVIDER=llm DECIDE_LLM_API_KEY=your_key go run .
```

| Method | Path            | Does                                                                                     |
|--------|-----------------|------------------------------------------------------------------------------------------|
| POST   | `/triage`       | Render `TicketInput` via `decide.StateTemplate` and evaluate `TriageDecision`            |
| POST   | `/triage/batch` | Batch-render and evaluate multiple tickets (uses `/v1/systemone/batch` natively on Laya) |

See `triage.http` for runnable requests.
