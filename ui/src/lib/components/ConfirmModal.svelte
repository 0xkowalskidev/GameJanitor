<script lang="ts">
  import { confirmState, resolveConfirm } from '$lib/stores/confirm';

  let state = $state({ open: false, title: '', message: '', confirmLabel: 'Confirm', danger: false, resolve: null as any });

  confirmState.subscribe((s) => { state = s; });

  function handleKeydown(e: KeyboardEvent) {
    if (!state.open) return;
    if (e.key === 'Escape') resolveConfirm(false);
    if (e.key === 'Enter') resolveConfirm(true);
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if state.open}
  <div class="backdrop" onclick={() => resolveConfirm(false)} role="presentation">
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
      <div class="modal-title">{state.title}</div>
      <div class="modal-message">{state.message}</div>
      <div class="modal-actions">
        <button class="btn-cancel" onclick={() => resolveConfirm(false)}>Cancel</button>
        <button
          class="btn-confirm"
          class:danger={state.danger}
          onclick={() => resolveConfirm(true)}
        >{state.confirmLabel}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: grid; place-items: center;
    z-index: 100;
    animation: fade-in 0.15s ease-out;
  }
  @keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }

  .modal {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    min-width: 340px;
    max-width: 440px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    animation: modal-in 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }
  @keyframes modal-in {
    from { opacity: 0; transform: scale(0.96) translateY(8px); }
    to { opacity: 1; transform: scale(1) translateY(0); }
  }

  .modal-title {
    font-size: 1rem; font-weight: 600;
    letter-spacing: -0.01em;
    margin-bottom: 8px;
  }
  .modal-message {
    font-size: 0.84rem; color: var(--text-secondary);
    line-height: 1.5;
    margin-bottom: 20px;
  }

  .modal-actions {
    display: flex; justify-content: flex-end; gap: 8px;
  }

  .btn-cancel {
    padding: 8px 16px; border-radius: var(--radius-sm);
    background: none; border: 1px solid var(--border-dim);
    color: var(--text-secondary); font-family: var(--font-body);
    font-size: 0.84rem; font-weight: 450; cursor: pointer;
    transition: border-color 0.15s, color 0.15s;
  }
  .btn-cancel:hover { border-color: var(--border); color: var(--text-primary); }

  .btn-confirm {
    padding: 8px 16px; border-radius: var(--radius-sm);
    background: var(--accent); border: none;
    color: #fff; font-family: var(--font-body);
    font-size: 0.84rem; font-weight: 520; cursor: pointer;
    transition: background 0.15s;
    box-shadow: 0 0 12px rgba(232, 114, 42, 0.2);
  }
  .btn-confirm:hover { background: var(--accent-hover); }
  .btn-confirm.danger {
    background: var(--danger);
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.2);
  }
  .btn-confirm.danger:hover { background: #dc2626; }
</style>
