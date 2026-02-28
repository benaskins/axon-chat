<script>
  import { onMount } from 'svelte';
  import { authenticatedFetch } from '$lib/auth';
  import MenuButton from '$lib/components/MenuButton.svelte';

  let agents = $state([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      const res = await authenticatedFetch('/api/agents');
      if (res.ok) {
        agents = await res.json();
      }
    } catch {}
    loading = false;
  });
</script>

<div class="home-menu">
  <MenuButton />
</div>
<div class="home">
  <h1>Aurelia Studio</h1>

  {#if loading}
    <div class="empty-state"><p>Loading agents...</p></div>
  {:else}
    <div class="agent-grid">
      {#each agents as agent}
        <div class="agent-card-wrapper">
          <a class="agent-card" href="/agents/{agent.slug}/conversations">
            {#if agent.base_image_url}
              <img class="agent-image" src={agent.base_image_url} alt={agent.name} />
            {:else}
              <div class="agent-placeholder">
                <span class="agent-emoji">{agent.avatar_emoji || '?'}</span>
              </div>
            {/if}
            <div class="agent-info">
              <span class="agent-name">{agent.name}</span>
              {#if agent.tagline}
                <span class="agent-tagline">{agent.tagline}</span>
              {/if}
            </div>
          </a>
          <a class="edit-link" href="/agents/{agent.slug}/edit" title="Edit {agent.name}">&#9998;</a>
        </div>
      {/each}
      <a class="agent-card add-card" href="/agents/new">
        <div class="agent-placeholder">
          <span class="add-icon">+</span>
        </div>
        <div class="agent-info">
          <span class="agent-name">New Agent</span>
        </div>
      </a>
    </div>
  {/if}
</div>

<style>
  .home-menu {
    position: fixed;
    top: 6px;
    right: 10px;
    z-index: 200;
  }

  .home {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 60px 20px;
    min-height: 100dvh;
  }

  h1 {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 40px;
  }

  .empty-state {
    color: var(--text-muted);
    font-size: 14px;
  }

  .agent-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    width: 100%;
    max-width: 720px;
  }

  .agent-card-wrapper {
    position: relative;
  }

  .agent-card {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-secondary);
    text-decoration: none;
    color: inherit;
    transition: border-color 0.15s;
    cursor: pointer;
    overflow: hidden;
    width: 100%;
  }

  .agent-card:hover {
    border-color: var(--accent);
  }

  .agent-image {
    width: 100%;
    aspect-ratio: 1;
    object-fit: cover;
    display: block;
  }

  .agent-placeholder {
    width: 100%;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
  }

  .agent-emoji {
    font-size: 48px;
    line-height: 1;
  }

  .add-icon {
    font-size: 36px;
    color: var(--text-muted);
    line-height: 1;
  }

  .agent-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px 12px;
  }

  .agent-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .agent-tagline {
    font-size: 12px;
    color: var(--text-muted);
    line-height: 1.3;
  }

  .edit-link {
    position: absolute;
    top: 8px;
    right: 8px;
    font-size: 14px;
    color: white;
    text-decoration: none;
    opacity: 0;
    transition: opacity 0.15s;
    padding: 2px 6px;
    border-radius: 4px;
    background: rgba(0, 0, 0, 0.5);
  }

  .edit-link:hover {
    color: var(--accent);
  }

  .agent-card-wrapper:hover .edit-link {
    opacity: 1;
  }

  .add-card {
    border-style: dashed;
    opacity: 0.6;
  }

  .add-card:hover {
    opacity: 1;
    border-color: var(--accent);
  }

  @media (max-width: 480px) {
    .home {
      padding: 40px 16px;
    }

    .agent-grid {
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;
    }

    .edit-link {
      opacity: 1;
    }
  }
</style>
<!-- invalidation-test -->
