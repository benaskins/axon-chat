<script>
  import MenuButton from './MenuButton.svelte';

  let { backHref = '', title, borderBottom = false, right } = $props();
</script>

<header class:border-bottom={borderBottom}>
  <div class="header-left">
    {#if backHref}
      <a class="back-link" href={backHref}>&larr;</a>
    {/if}
    <h1>{title}</h1>
  </div>
  <div class="header-actions">
    {#if right}
      <div class="header-right">
        {@render right()}
      </div>
    {/if}
    <MenuButton />
  </div>
</header>

<style>
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 24px;
    position: sticky;
    top: 0;
    z-index: 10;
    background: var(--bg-primary);
    padding-bottom: 8px;
  }

  header.border-bottom {
    margin-bottom: 0;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .back-link {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 18px;
    line-height: 1;
    padding: 8px;
    margin: -8px;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  header h1 {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  @media (max-width: 480px) {
    header.border-bottom h1 {
      font-size: 14px;
    }

    .header-right {
      display: none;
    }
  }
</style>
