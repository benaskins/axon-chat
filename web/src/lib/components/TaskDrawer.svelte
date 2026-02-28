<script>
  import { authenticatedFetch } from '$lib/auth';
  import BlurredImage from './BlurredImage.svelte';

  let { open = false, agentSlug = '', onclose } = $props();

  let activeTab = $state('tasks');
  let tasks = $state([]);
  let images = $state([]);
  let tasksLoading = $state(false);
  let imagesLoading = $state(false);
  let lightboxUrl = $state(null);
  let selectedImage = $state(null);

  $effect(() => {
    if (open && agentSlug) {
      if (activeTab === 'tasks') {
        loadTasks();
      } else {
        loadImages();
      }
    }
  });

  async function loadTasks() {
    tasksLoading = true;
    try {
      const res = await authenticatedFetch(`/api/tasks?agent=${agentSlug}`);
      if (res.ok) {
        tasks = await res.json();
      }
    } catch {}
    tasksLoading = false;
  }

  async function loadImages() {
    imagesLoading = true;
    try {
      const res = await authenticatedFetch(`/api/agents/${agentSlug}/gallery`);
      if (res.ok) {
        const data = await res.json();
        images = data.images || [];
      }
    } catch {}
    imagesLoading = false;
  }

  function switchTab(tab) {
    activeTab = tab;
    if (tab === 'tasks' && tasks.length === 0) loadTasks();
    if (tab === 'gallery' && images.length === 0) loadImages();
  }

  function statusIcon(status) {
    if (status === 'completed') return '\u2713';
    if (status === 'failed') return '\u2717';
    if (status === 'running') return '\u25CF';
    return '\u25CB'; // queued
  }

  function timeAgo(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);
    if (seconds < 60) return 'just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
    return date.toLocaleDateString();
  }

  function taskTypeLabel(type) {
    if (type === 'image_generation') return 'Image';
    if (type === 'claude_session') return 'Code';
    return type || 'Task';
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') onclose?.();
  }

  function openImageLightbox(image) {
    selectedImage = image;
    lightboxUrl = image.url;
  }

  function closeLightbox() {
    selectedImage = null;
    lightboxUrl = null;
  }

  async function setAsBase(imageId) {
    try {
      const res = await authenticatedFetch(`/api/agents/${agentSlug}/gallery/${imageId}/base`, {
        method: 'PUT'
      });
      if (res.ok) {
        images = images.map(img => ({ ...img, is_base: img.id === imageId }));
        if (selectedImage) {
          selectedImage = { ...selectedImage, is_base: true };
        }
      }
    } catch {}
  }

  /** Called from parent to update task list with real-time event data. */
  export function updateTask(taskData) {
    const idx = tasks.findIndex(t => t.task_id === taskData.task_id);
    if (idx >= 0) {
      tasks[idx] = { ...tasks[idx], ...taskData };
      tasks = tasks;
    } else {
      // New task — prepend
      tasks = [taskData, ...tasks];
    }
  }

  /** Called from parent when a new image completes. */
  export function addImage(imageData) {
    // Reload gallery to get full metadata
    if (open && activeTab === 'gallery') {
      loadImages();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="drawer-backdrop" onclick={onclose} role="presentation"></div>
  <aside class="drawer">
    <div class="drawer-header">
      <div class="tabs">
        <button class="tab" class:active={activeTab === 'tasks'} onclick={() => switchTab('tasks')}>
          Tasks
        </button>
        <button class="tab" class:active={activeTab === 'gallery'} onclick={() => switchTab('gallery')}>
          Gallery
        </button>
      </div>
      <button class="close-btn" onclick={onclose}>&times;</button>
    </div>

    <div class="drawer-content">
      {#if activeTab === 'tasks'}
        {#if tasksLoading && tasks.length === 0}
          <div class="empty-state">Loading tasks...</div>
        {:else if tasks.length === 0}
          <div class="empty-state">No tasks yet.</div>
        {:else}
          <div class="task-list">
            {#each tasks as task}
              <details class="task-item {task.status}">
                <summary>
                  <span class="task-status-icon {task.status}">{statusIcon(task.status)}</span>
                  <span class="task-type">{taskTypeLabel(task.type)}</span>
                  <span class="task-desc">{task.description || 'No description'}</span>
                  <span class="task-time">{timeAgo(task.created_at)}</span>
                </summary>
                <div class="task-details">
                  {#if task.result_summary}
                    <div class="detail-section">
                      <div class="detail-label">Result</div>
                      <div class="detail-text">{task.result_summary}</div>
                    </div>
                  {/if}
                  {#if task.error}
                    <div class="detail-section">
                      <div class="detail-label error">Error</div>
                      <div class="detail-text error">{task.error}</div>
                    </div>
                  {/if}
                  {#if task.artifact_id && task.type === 'image_generation'}
                    <div class="detail-section">
                      <a class="detail-link" href="/api/images/{task.artifact_id}" target="_blank">
                        View image
                      </a>
                    </div>
                  {/if}
                  <div class="detail-meta">
                    ID: {task.task_id}
                    {#if task.completed_at}
                      &middot; Completed {timeAgo(task.completed_at)}
                    {/if}
                  </div>
                </div>
              </details>
            {/each}
          </div>
        {/if}

      {:else}
        {#if imagesLoading && images.length === 0}
          <div class="empty-state">Loading gallery...</div>
        {:else if images.length === 0}
          <div class="empty-state">No images yet.</div>
        {:else}
          <div class="gallery-grid">
            {#each images as image}
              <button
                class="gallery-item"
                class:is-base={image.is_base}
                onclick={() => openImageLightbox(image)}
              >
                <BlurredImage
                  src={image.thumbnail_url || image.url}
                  isNsfw={image.nsfw_detected}
                  alt={image.prompt}
                />
                {#if image.is_base}
                  <div class="base-badge">Base</div>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  </aside>
{/if}

{#if lightboxUrl && selectedImage}
  <div class="lightbox" onclick={closeLightbox} role="button" tabindex="-1">
    <div class="lightbox-content" onclick={(e) => e.stopPropagation()}>
      <button class="lightbox-close" onclick={closeLightbox}>&times;</button>
      <BlurredImage
        src={selectedImage.url}
        isNsfw={selectedImage.nsfw_detected}
        alt={selectedImage.prompt}
      />
      <div class="image-info">
        {#if selectedImage.prompt}
          <div class="prompt"><strong>Prompt:</strong> {selectedImage.prompt}</div>
        {/if}
        <div class="meta">
          {#if selectedImage.model}<span>{selectedImage.model}</span><span>&middot;</span>{/if}
          <span>{timeAgo(selectedImage.created_at)}</span>
        </div>
        {#if !selectedImage.is_base}
          <button class="set-base-btn" onclick={() => setAsBase(selectedImage.id)}>
            Set as Base Image
          </button>
        {:else}
          <div class="current-base">Current base image</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 50;
  }

  .drawer {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(400px, 90vw);
    background: var(--bg-secondary);
    border-left: 1px solid var(--border);
    z-index: 51;
    display: flex;
    flex-direction: column;
    animation: slide-in 0.2s ease-out;
  }

  @keyframes slide-in {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
  }

  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .tabs {
    display: flex;
    gap: 4px;
  }

  .tab {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
  }

  .tab.active {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .tab:hover:not(.active) {
    color: var(--text-secondary);
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 24px;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }

  .close-btn:hover {
    color: var(--text-primary);
  }

  .drawer-content {
    flex: 1;
    overflow-y: auto;
    padding: 12px;
  }

  .empty-state {
    text-align: center;
    padding: 40px 20px;
    color: var(--text-muted);
    font-size: 13px;
  }

  /* Tasks */
  .task-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .task-item {
    border: 1px solid var(--border);
    border-radius: 8px;
    overflow: hidden;
  }

  .task-item summary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    cursor: pointer;
    font-size: 13px;
    list-style: none;
  }

  .task-item summary::-webkit-details-marker {
    display: none;
  }

  .task-status-icon {
    flex-shrink: 0;
    font-size: 12px;
    width: 16px;
    text-align: center;
  }

  .task-status-icon.completed { color: var(--accent); }
  .task-status-icon.failed { color: var(--error, #e55); }
  .task-status-icon.running {
    color: var(--accent);
    animation: pulse 1.5s ease-in-out infinite;
  }
  .task-status-icon.queued { color: var(--text-muted); }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  .task-type {
    flex-shrink: 0;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 3px;
  }

  .task-desc {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-secondary);
  }

  .task-time {
    flex-shrink: 0;
    font-size: 11px;
    color: var(--text-muted);
  }

  .task-details {
    padding: 8px 12px 12px;
    border-top: 1px solid var(--border);
    font-size: 12px;
  }

  .detail-section {
    margin-bottom: 8px;
  }

  .detail-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
    margin-bottom: 2px;
  }

  .detail-label.error { color: var(--error, #e55); }

  .detail-text {
    color: var(--text-secondary);
    white-space: pre-wrap;
    line-height: 1.4;
  }

  .detail-text.error { color: var(--error, #e55); }

  .detail-link {
    color: var(--accent);
    text-decoration: none;
    font-size: 12px;
  }

  .detail-link:hover {
    text-decoration: underline;
  }

  .detail-meta {
    font-size: 10px;
    color: var(--text-muted);
    margin-top: 8px;
  }

  /* Gallery */
  .gallery-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 8px;
  }

  .gallery-item {
    position: relative;
    aspect-ratio: 1;
    border-radius: 6px;
    overflow: hidden;
    background: var(--bg-tertiary);
    border: 2px solid var(--border);
    cursor: pointer;
    padding: 0;
    transition: border-color 0.15s;
  }

  .gallery-item:hover {
    border-color: var(--accent);
  }

  .gallery-item.is-base {
    border-color: var(--accent);
  }

  .gallery-item img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .base-badge {
    position: absolute;
    top: 4px;
    right: 4px;
    background: var(--accent);
    color: white;
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 10px;
    font-weight: 600;
  }

  /* Lightbox */
  .lightbox {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.9);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 20px;
  }

  .lightbox-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .lightbox-close {
    position: absolute;
    top: -36px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 32px;
    cursor: pointer;
    line-height: 1;
    padding: 0;
  }

  .lightbox-close:hover { opacity: 0.7; }

  .lightbox-content img {
    max-width: 100%;
    max-height: 60vh;
    object-fit: contain;
    border-radius: 6px;
  }

  .image-info {
    background: var(--bg-secondary);
    padding: 12px;
    border-radius: 6px;
    max-width: 500px;
  }

  .prompt {
    font-size: 13px;
    margin-bottom: 6px;
    line-height: 1.4;
  }

  .prompt strong {
    color: var(--text-muted);
    font-weight: 500;
  }

  .meta {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 8px;
  }

  .set-base-btn {
    width: 100%;
    padding: 8px;
    background: var(--accent);
    border: none;
    color: white;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
  }

  .set-base-btn:hover { opacity: 0.9; }

  .current-base {
    text-align: center;
    padding: 8px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    font-size: 12px;
    color: var(--text-muted);
  }

  @media (max-width: 480px) {
    .drawer {
      width: 100vw;
    }
  }
</style>
