<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { authenticatedFetch } from '$lib/auth';
  import { setMenuItems, clearMenuItems } from '$lib/stores/menu.js';
  import Header from '$lib/components/Header.svelte';

  let agent = $state(null);
  let conversations = $state([]);
  let loading = $state(true);
  let creating = $state(false);

  let visibleConversations = $derived(
    conversations.filter(c => c.message_count > 0)
  );

  onMount(async () => {
    const slug = $page.params.slug;

    try {
      const [agentRes, convosRes] = await Promise.all([
        authenticatedFetch(`/api/agents/${slug}`),
        authenticatedFetch(`/api/agents/${slug}/conversations`)
      ]);

      if (!agentRes.ok) {
        goto('/');
        return;
      }

      agent = await agentRes.json();
      if (convosRes.ok) {
        conversations = await convosRes.json();
      }

      setMenuItems([
        { type: 'link', label: 'Edit Agent', href: `/agents/${agent.slug}/edit?from=/agents/${agent.slug}/conversations` },
      ]);
    } catch {
      goto('/');
    }

    loading = false;
  });

  onDestroy(() => {
    clearMenuItems();
  });

  async function newConversation() {
    if (creating) return;
    creating = true;

    try {
      const res = await authenticatedFetch(`/api/agents/${$page.params.slug}/conversations`, {
        method: 'POST'
      });
      if (res.ok) {
        const conv = await res.json();
        goto(`/chat/${$page.params.slug}/${conv.id}`);
      }
    } catch (err) {
      creating = false;
    }
  }

  async function deleteConversation(event, convId) {
    event.preventDefault();
    event.stopPropagation();

    if (!confirm('Delete this conversation?')) return;

    try {
      const res = await authenticatedFetch(`/api/conversations/${convId}`, {
        method: 'DELETE'
      });
      if (res.ok) {
        conversations = conversations.filter(c => c.id !== convId);
      }
    } catch {}
  }

  function timeAgo(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);

    if (seconds < 60) return 'just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
    return date.toLocaleDateString();
  }
</script>

{#if loading}
  <div class="container"><p class="loading">Loading...</p></div>
{:else if agent}
<div class="container">
  <Header backHref="/" title="{agent.avatar_emoji} {agent.name}" />

  <button class="new-btn" onclick={newConversation} disabled={creating}>
    {creating ? 'Creating...' : 'New Conversation'}
  </button>

  {#if visibleConversations.length === 0}
    <div class="empty-state">
      <p>No conversations yet.</p>
      <p class="hint">Start one to begin chatting with {agent.name}.</p>
    </div>
  {:else}
    <div class="conversation-list">
      {#each visibleConversations as conv}
        <div class="conversation-row-wrapper">
          <a class="conversation-row" href="/chat/{agent.slug}/{conv.id}">
            <span class="conv-title">{conv.title || 'Untitled'}</span>
            <span class="conv-meta">
              {conv.message_count} messages &middot; {timeAgo(conv.updated_at)}
            </span>
          </a>
          <button class="delete-btn" onclick={(e) => deleteConversation(e, conv.id)} title="Delete conversation">&times;</button>
        </div>
      {/each}
    </div>
  {/if}
</div>
{/if}

<style>
  .container {
    max-width: 640px;
    margin: 0 auto;
    padding: 20px;
    min-height: 100dvh;
  }

  .loading {
    color: var(--text-muted);
    text-align: center;
    padding: 60px 0;
  }

  .new-btn {
    width: 100%;
    padding: 12px;
    background: var(--accent);
    border: none;
    color: white;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    margin-bottom: 16px;
  }

  .new-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .new-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .empty-state {
    text-align: center;
    padding: 40px 0;
    color: var(--text-muted);
    font-size: 14px;
  }

  .hint {
    font-size: 13px;
    margin-top: 4px;
  }

  .conversation-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .conversation-row-wrapper {
    position: relative;
  }

  .conversation-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 12px 14px;
    border: 1px solid var(--border);
    border-radius: 8px;
    text-decoration: none;
    color: inherit;
    transition: border-color 0.15s;
  }

  .conversation-row:hover {
    border-color: var(--accent);
  }

  .delete-btn {
    position: absolute;
    top: 8px;
    right: 8px;
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    line-height: 1;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .conversation-row-wrapper:hover .delete-btn {
    opacity: 1;
  }

  .delete-btn:hover {
    color: #e55;
  }

  .conv-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .conv-meta {
    font-size: 12px;
    color: var(--text-muted);
  }

  @media (max-width: 480px) {
    .delete-btn {
      opacity: 1;
    }
  }
</style>
