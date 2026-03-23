import { writable } from 'svelte/store';

export interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
  persistent: boolean;
}

let nextId = 0;

export const toasts = writable<Toast[]>([]);

export function toast(message: string, type: Toast['type'] = 'info') {
  const id = nextId++;
  const persistent = type === 'error';

  toasts.update(t => [...t, { id, message, type, persistent }]);

  if (!persistent) {
    setTimeout(() => dismiss(id), 5000);
  }
}

export function dismiss(id: number) {
  toasts.update(t => t.filter(toast => toast.id !== id));
}
