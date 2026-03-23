import { writable } from 'svelte/store';

interface ConfirmState {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  danger: boolean;
  inputMode: boolean;
  inputPlaceholder: string;
  inputValue: string;
  resolve: ((value: boolean | string | null) => void) | null;
}

const initial: ConfirmState = {
  open: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  danger: false,
  inputMode: false,
  inputPlaceholder: '',
  inputValue: '',
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
      ...initial,
      open: true,
      title: opts.title,
      message: opts.message,
      confirmLabel: opts.confirmLabel || 'Confirm',
      danger: opts.danger ?? false,
      resolve: (v) => resolve(!!v),
    });
  });
}

export function prompt(opts: {
  title: string;
  message?: string;
  placeholder?: string;
  confirmLabel?: string;
}): Promise<string | null> {
  return new Promise((resolve) => {
    confirmState.set({
      ...initial,
      open: true,
      title: opts.title,
      message: opts.message || '',
      confirmLabel: opts.confirmLabel || 'Create',
      inputMode: true,
      inputPlaceholder: opts.placeholder || '',
      inputValue: '',
      resolve: (v) => resolve(typeof v === 'string' ? v : null),
    });
  });
}

export function resolveConfirm(accepted: boolean, inputValue?: string) {
  confirmState.update((s) => {
    if (accepted && s.inputMode) {
      s.resolve?.(inputValue || '');
    } else {
      s.resolve?.(accepted ? true : null);
    }
    return initial;
  });
}
