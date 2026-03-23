<script lang="ts">
  import '../styles/tokens.css';
  import { onMount, onDestroy } from 'svelte';
  import { ToastContainer } from '$lib/components';
  import { connect, disconnect, enableAutoToasts, initAuth } from '$lib/stores';

  let { children } = $props();

  onMount(() => {
    initAuth();
    connect();
    enableAutoToasts();
  });

  onDestroy(() => {
    disconnect();
  });
</script>

<nav>
  <div class="n-left">
    <a href="/" class="brand">
      <div class="brand-mark">
        <svg viewBox="0 0 16 16"><path d="M3 2.5A1.5 1.5 0 0 1 4.5 1h7A1.5 1.5 0 0 1 13 2.5v11a1.5 1.5 0 0 1-1.5 1.5h-7A1.5 1.5 0 0 1 3 13.5v-11zM6 4h4v1H6V4zm0 3h4v1H6V7zm0 3h3v1H6v-1z"/></svg>
      </div>
      Gamejanitor
    </a>
    <div class="n-links">
      <a href="/">Dashboard</a>
      <a href="/settings">Settings</a>
    </div>
  </div>
  <div class="n-right">
    <div class="auth"><span class="auth-dot"></span>Local</div>
  </div>
</nav>

{@render children()}

<ToastContainer />
