<script>
  import { onMount, onDestroy } from 'svelte';
  import { logout } from '$lib/auth';
  import { menuItems } from '$lib/stores/menu.js';

  let open = $state(false);
  let items = $state([]);
  let menuRef;

  const unsubscribe = menuItems.subscribe(v => items = v);

  onMount(() => {
    document.addEventListener('mousedown', handleOutsideClick);
  });

  onDestroy(() => {
    unsubscribe();
    if (typeof document !== 'undefined') {
      document.removeEventListener('mousedown', handleOutsideClick);
    }
  });

  function handleOutsideClick(e) {
    if (open && menuRef && !menuRef.contains(e.target)) {
      open = false;
    }
  }

  function closeAndRun(fn) {
    return () => {
      open = false;
      fn();
    };
  }
</script>

<div class="menu-wrapper" bind:this={menuRef}>
  <button class="hamburger" class:open onclick={() => open = !open} aria-label="Menu">
    <span class="bar"></span>
    <span class="bar"></span>
    <span class="bar"></span>
  </button>

  {#if open}
    <div class="dropdown">
      {#each items as item}
        {#if item.type === 'link'}
          <a class="menu-item" href={item.href} onclick={() => open = false}>
            {item.label}
          </a>
        {:else if item.type === 'button'}
          <button class="menu-item" onclick={closeAndRun(item.onclick)} disabled={item.disabled}>
            {item.label}
          </button>
        {:else if item.type === 'toggle'}
          <button class="menu-item" onclick={() => item.onclick()} disabled={item.disabled}>
            <span class="menu-label">{item.label}</span>
            <span class="menu-value" class:active={item.active}>{item.active ? 'On' : 'Off'}</span>
          </button>
        {:else if item.type === 'select'}
          <label class="menu-item">
            <span class="menu-label">{item.label}</span>
            <select class="menu-select" value={item.value}
              onchange={(e) => item.onchange(e.target.value)}
              disabled={item.disabled}>
              {#each item.options as opt}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          </label>
        {/if}
      {/each}

      {#if items.length > 0}
        <div class="menu-divider"></div>
      {/if}

      <a class="menu-item" href="/" onclick={() => open = false}>Home</a>
      <a class="menu-item" href="/agents/new" onclick={() => open = false}>New Agent</a>
      <button class="menu-item" onclick={closeAndRun(logout)}>Sign out</button>
    </div>
  {/if}
</div>

<style>
  .menu-wrapper {
    position: relative;
  }

  .hamburger {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 5px;
    width: 36px;
    height: 36px;
    padding: 8px;
    background: none;
    border: none;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .bar {
    display: block;
    width: 100%;
    height: 2px;
    background: var(--text-muted);
    border-radius: 1px;
    transition: transform 0.2s, opacity 0.2s;
    transform-origin: center;
  }

  .hamburger:hover .bar {
    background: var(--text-secondary);
  }

  .hamburger.open .bar:nth-child(1) {
    transform: translateY(7px) rotate(45deg);
  }

  .hamburger.open .bar:nth-child(2) {
    opacity: 0;
  }

  .hamburger.open .bar:nth-child(3) {
    transform: translateY(-7px) rotate(-45deg);
  }

  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 6px;
    min-width: 220px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    animation: dropdown-in 0.15s ease-out;
    z-index: 200;
  }

  @keyframes dropdown-in {
    from {
      opacity: 0;
      transform: scale(0.95) translateY(-4px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .menu-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
    font-family: var(--font-sans);
    cursor: pointer;
    text-align: left;
    text-decoration: none;
    min-height: 44px;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .menu-item:hover {
    background: var(--bg-tertiary);
  }

  .menu-item:active {
    background: var(--bg-tertiary);
  }

  .menu-item:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .menu-label {
    color: var(--text-secondary);
  }

  .menu-value {
    color: var(--text-muted);
    font-size: 13px;
  }

  .menu-value.active {
    color: var(--accent);
  }

  .menu-select {
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 13px;
    outline: none;
    cursor: pointer;
    max-width: 140px;
    -webkit-appearance: none;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='8' height='5' viewBox='0 0 8 5'%3E%3Cpath fill='%23888' d='M0 0l4 5 4-5z'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 6px center;
    padding-right: 20px;
  }

  .menu-select:focus {
    border-color: var(--accent);
  }

  .menu-divider {
    height: 1px;
    background: var(--border);
    margin: 4px 8px;
  }
</style>
