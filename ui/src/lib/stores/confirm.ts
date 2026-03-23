import { writable } from 'svelte/store';

interface ConfirmState {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  danger: boolean;
  resolve: ((value: boolean) => void) | null;
}

const initial: ConfirmState = {
  open: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  danger: false,
  resolve: null,
};

export const confirmState = writable<ConfirmState>(initial);

export function confirm(opts: {
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
}): Promise<boolean> {
  return new Promise((resolve) => {
    confirmState.set({
      open: true,
      title: opts.title,
      message: opts.message,
      confirmLabel: opts.confirmLabel || 'Confirm',
      danger: opts.danger ?? false,
      resolve,
    });
  });
}

export function resolveConfirm(accepted: boolean) {
  confirmState.update((s) => {
    s.resolve?.(accepted);
    return initial;
  });
}
