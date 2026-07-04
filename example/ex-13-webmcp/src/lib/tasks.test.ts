import { describe, expect, it } from 'vitest';
import { createTaskStore } from './tasks';

describe('createTaskStore', () => {
  it('adds and lists tasks', () => {
    const store = createTaskStore();
    const task = store.add('Ship WebMCP demo');
    expect(task.title).toBe('Ship WebMCP demo');
    expect(store.list()).toHaveLength(1);
  });

  it('filters by status', () => {
    const store = createTaskStore();
    const a = store.add('One');
    store.toggle(a.id);
    store.add('Two');

    expect(store.list('open')).toHaveLength(1);
    expect(store.list('done')).toHaveLength(1);
  });

  it('toggles and removes tasks', () => {
    const store = createTaskStore();
    const task = store.add('Toggle me');
    const toggled = store.toggle(task.id);
    expect(toggled.status).toBe('done');

    store.remove(task.id);
    expect(store.list()).toHaveLength(0);
  });

  it('reports stats', () => {
    const store = createTaskStore();
    const a = store.add('A');
    store.add('B');
    store.toggle(a.id);

    expect(store.stats()).toEqual({ total: 2, open: 1, done: 1 });
  });

  it('rejects empty titles', () => {
    const store = createTaskStore();
    expect(() => store.add('   ')).toThrow(/title is required/);
  });
});
