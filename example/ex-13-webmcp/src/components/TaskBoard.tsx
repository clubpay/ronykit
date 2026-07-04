import { useMemo, useState, type FormEvent } from 'react';
import { useWebMCP } from '@mcp-b/react-webmcp';
import { z } from 'zod';

import {
  createTaskStore,
  summarizeTasks,
  type TaskFilter,
  type TaskStore,
} from '../lib/tasks';

const statusSchema = z.enum(['all', 'open', 'done']);

interface TaskBoardProps {
  initialStore?: TaskStore;
}

export function TaskBoard({ initialStore }: TaskBoardProps) {
  const store = useMemo(() => initialStore ?? createTaskStore(), [initialStore]);
  const [filter, setFilter] = useState<TaskFilter>('all');
  const [title, setTitle] = useState('');
  const [tasks, setTasks] = useState(() => store.list('all'));
  const [agentBusy, setAgentBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = (nextFilter: TaskFilter = filter) => {
    setTasks(store.list(nextFilter));
    setError(null);
  };

  const run = async <T,>(fn: () => T | Promise<T>): Promise<T> => {
    setAgentBusy(true);
    try {
      const result = await fn();
      refresh();
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'operation failed';
      setError(message);
      throw err;
    } finally {
      setAgentBusy(false);
    }
  };

  useWebMCP(
    {
      name: 'tasks_list',
      description:
        'List tasks on the board. Optionally filter by status (all, open, done). Returns task summaries.',
      inputSchema: {
        status: statusSchema
          .optional()
          .describe('Filter: all (default), open, or done'),
      },
      annotations: { title: 'List tasks', readOnlyHint: true },
      handler: async (input) => {
        const status = input.status ?? 'all';
        const items = store.list(status);
        return { status, total: items.length, items: summarizeTasks(items) };
      },
    },
    [filter, tasks],
  );

  useWebMCP(
    {
      name: 'tasks_add',
      description: 'Add a new open task by title. Returns the created task.',
      inputSchema: {
        title: z.string().min(1).describe('Task title (non-empty)'),
      },
      annotations: { title: 'Add task', readOnlyHint: false },
      handler: async (input) =>
        run(() => {
          const task = store.add(input.title);
          return { success: true, task: summarizeTasks([task])[0] };
        }),
    },
    [filter, tasks],
  );

  useWebMCP(
    {
      name: 'tasks_toggle',
      description:
        'Toggle a task between open and done by id. Returns the updated task.',
      inputSchema: {
        id: z.string().describe('Task id (e.g. task_1)'),
      },
      annotations: { title: 'Toggle task', readOnlyHint: false },
      handler: async (input) =>
        run(() => {
          const task = store.toggle(input.id);
          return { success: true, task: summarizeTasks([task])[0] };
        }),
    },
    [filter, tasks],
  );

  useWebMCP(
    {
      name: 'tasks_remove',
      description: 'Remove a task by id. Returns the deleted task summary.',
      inputSchema: {
        id: z.string().describe('Task id to delete'),
      },
      annotations: { title: 'Remove task', readOnlyHint: false },
      handler: async (input) =>
        run(() => {
          const task = store.remove(input.id);
          return { success: true, removed: summarizeTasks([task])[0] };
        }),
    },
    [filter, tasks],
  );

  useWebMCP(
    {
      name: 'tasks_stats',
      description:
        'Return counts of total, open, and done tasks for the current board.',
      inputSchema: {},
      annotations: { title: 'Task stats', readOnlyHint: true },
      handler: async () => {
        const counts = store.stats();
        return { ...counts, filter };
      },
    },
    [filter, tasks],
  );

  const onAdd = (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim()) return;
    run(() => store.add(title));
    setTitle('');
  };

  return (
    <div className="board">
      <header className="board__header">
        <div>
          <h1>Task board</h1>
          <p className="muted">
            Human UI and WebMCP tools share <code>src/lib/tasks.ts</code>.
          </p>
        </div>
        {agentBusy && <span className="badge">Agent updating…</span>}
      </header>

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}

      <form className="add-form" onSubmit={onAdd}>
        <label htmlFor="title">New task</label>
        <div className="add-form__row">
          <input
            id="title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="What needs doing?"
          />
          <button type="submit">Add</button>
        </div>
      </form>

      <div className="filters" role="tablist" aria-label="Filter tasks">
        {(['all', 'open', 'done'] as const).map((value) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={filter === value}
            className={filter === value ? 'active' : undefined}
            onClick={() => {
              setFilter(value);
              refresh(value);
            }}
          >
            {value}
          </button>
        ))}
      </div>

      <ul className="task-list">
        {tasks.length === 0 && <li className="muted">No tasks yet.</li>}
        {tasks.map((task) => (
          <li key={task.id}>
            <label>
              <input
                type="checkbox"
                checked={task.status === 'done'}
                onChange={() => run(() => store.toggle(task.id))}
              />
              <span className={task.status === 'done' ? 'done' : undefined}>
                {task.title}
              </span>
            </label>
            <button
              type="button"
              className="ghost"
              onClick={() => run(() => store.remove(task.id))}
            >
              Remove
            </button>
          </li>
        ))}
      </ul>

      <footer className="stats muted">
        {store.stats().open} open · {store.stats().done} done ·{' '}
        {store.stats().total} total
      </footer>
    </div>
  );
}
