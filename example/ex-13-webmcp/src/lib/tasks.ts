export type TaskStatus = 'open' | 'done';

export interface Task {
  id: string;
  title: string;
  status: TaskStatus;
  createdAt: string;
}

export interface TaskStats {
  total: number;
  open: number;
  done: number;
}

export type TaskFilter = 'all' | TaskStatus;

let seq = 0;

function nextId(): string {
  seq += 1;
  return `task_${seq}`;
}

export function createTaskStore(initial: Task[] = []) {
  let tasks = [...initial];

  const list = (filter: TaskFilter = 'all'): Task[] => {
    const sorted = [...tasks].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    if (filter === 'all') return sorted;
    return sorted.filter((t) => t.status === filter);
  };

  const add = (title: string): Task => {
    const trimmed = title.trim();
    if (!trimmed) {
      throw new Error('title is required');
    }

    const task: Task = {
      id: nextId(),
      title: trimmed,
      status: 'open',
      createdAt: new Date().toISOString(),
    };
    tasks = [...tasks, task];
    return task;
  };

  const toggle = (id: string): Task => {
    const idx = tasks.findIndex((t) => t.id === id);
    if (idx === -1) {
      throw new Error(`task not found: ${id}`);
    }

    const current = tasks[idx];
    const updated: Task = {
      ...current,
      status: current.status === 'open' ? 'done' : 'open',
    };
    tasks = tasks.map((t) => (t.id === id ? updated : t));
    return updated;
  };

  const remove = (id: string): Task => {
    const task = tasks.find((t) => t.id === id);
    if (!task) {
      throw new Error(`task not found: ${id}`);
    }
    tasks = tasks.filter((t) => t.id !== id);
    return task;
  };

  const stats = (): TaskStats => {
    const open = tasks.filter((t) => t.status === 'open').length;
    const done = tasks.filter((t) => t.status === 'done').length;
    return { total: tasks.length, open, done };
  };

  return { list, add, toggle, remove, stats };
}

export type TaskStore = ReturnType<typeof createTaskStore>;

export function summarizeTasks(items: Task[]) {
  return items.map((t) => ({
    id: t.id,
    title: t.title,
    status: t.status,
  }));
}
