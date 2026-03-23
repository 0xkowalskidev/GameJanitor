<script lang="ts">
  import { toasts, dismiss } from '$lib/stores/toasts';
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as t (t.id)}
      <div class="toast {t.type}" role="alert">
        <span class="toast-msg">{t.message}</span>
        <button class="toast-close" onclick={() => dismiss(t.id)}>×</button>
      </div>
    {/each}
  </div>
{/if}

<style>
  .toast-container {
    position: fixed;
    bottom: 20px; right: 20px;
    z-index: 100;
    display: flex; flex-direction: column-reverse;
    gap: 8px;
    max-width: 380px;
  }

  .toast {
    display: flex; align-items: center; justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    font-size: 0.82rem;
    animation: slide-in 0.25s ease-out;
    box-shadow: 0 4px 16px rgba(0,0,0,0.3);
  }

  .toast.success { background: rgba(34,197,94,0.12); border: 1px solid rgba(34,197,94,0.2); color: var(--live); }
  .toast.error { background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.2); color: var(--danger); }
  .toast.info { background: var(--bg-elevated); border: 1px solid var(--border); color: var(--text-secondary); }

  .toast-msg { flex: 1; }

  .toast-close {
    background: none; border: none; color: inherit;
    font-size: 1.1rem; cursor: pointer; opacity: 0.6;
    padding: 0 2px; line-height: 1;
    transition: opacity 0.15s;
  }
  .toast-close:hover { opacity: 1; }

  @keyframes slide-in {
    from { opacity: 0; transform: translateX(20px); }
    to { opacity: 1; transform: translateX(0); }
  }
</style>
