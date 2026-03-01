<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { authenticatedFetch } from '$lib/auth';
  import Header from '$lib/components/Header.svelte';

  let isNew = false;
  let loading = $state(true);
  let saving = $state(false);
  let error = $state('');
  let models = $state([]);
  let showPreview = $state(false);
  let confirmDelete = $state(false);

  let returnTo = $state('/');
  let slug = $state('');
  let name = $state('');
  let tagline = $state('');
  let avatarEmoji = $state('');
  let systemPrompt = $state('');
  let constraints = $state('');
  let greeting = $state('');
  let defaultModel = $state('');
  let temperature = $state(0.7);
  let thinkEnabled = $state(true);
  let topP = $state(null);
  let topK = $state(null);
  let minP = $state(null);
  let presencePenalty = $state(null);
  let maxTokens = $state(null);
  let activeTab = $state('persona');
  let enabledSkills = $state(new Set());

  // Snapshot for dirty tracking
  let snapshot = $state(null);

  let dirty = $derived(snapshot !== null && JSON.stringify(currentValues()) !== snapshot);

  function currentValues() {
    return {
      name, tagline, avatarEmoji, systemPrompt, constraints,
      greeting, defaultModel, temperature, thinkEnabled, topP, topK, minP,
      presencePenalty, maxTokens, enabledSkills: [...enabledSkills].sort(),
    };
  }

  function takeSnapshot() {
    snapshot = JSON.stringify(currentValues());
  }

  let availableSkills = $state([]);

  // Autogrow action for textareas
  function autogrow(node) {
    function resize() {
      node.style.height = 'auto';
      node.style.height = node.scrollHeight + 'px';
    }
    resize();
    node.addEventListener('input', resize);
    return {
      destroy() {
        node.removeEventListener('input', resize);
      }
    };
  }

  function slugify(text) {
    return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  }

  function buildPreview() {
    const parts = [];
    if (systemPrompt.trim()) parts.push(systemPrompt.trim());
    if (constraints.trim()) parts.push(`## Constraints\n${constraints.trim()}`);
    return parts.join('\n\n');
  }

  // Cmd+S / Ctrl+S handler
  function handleKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      if (!saving && name.trim()) save();
    }
  }

  // Beforeunload handler
  function handleBeforeunload(e) {
    if (dirty) {
      e.preventDefault();
      e.returnValue = '';
    }
  }

  onMount(async () => {
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('beforeunload', handleBeforeunload);

    const fromParam = $page.url.searchParams.get('from');
    if (fromParam) returnTo = fromParam;

    const slugParam = $page.params.slug;
    isNew = !slugParam;

    const [modelsRes, skillsRes] = await Promise.all([
      authenticatedFetch('/api/models'),
      fetch('/api/skills'),
    ]);
    if (modelsRes.ok) {
      models = await modelsRes.json();
    }
    if (skillsRes.ok) {
      availableSkills = await skillsRes.json();
    }

    if (!isNew) {
      try {
        const res = await authenticatedFetch(`/api/agents/${slugParam}`);
        if (!res.ok) {
          goto('/');
          return;
        }
        const agent = await res.json();
        slug = agent.slug;
        name = agent.name;
        tagline = agent.tagline || '';
        avatarEmoji = agent.avatar_emoji || '';
        systemPrompt = agent.system_prompt || '';
        constraints = agent.constraints || '';
        greeting = agent.greeting || '';
        defaultModel = agent.default_model || '';
        temperature = agent.temperature ?? 0.7;
        thinkEnabled = agent.think ?? true;
        topP = agent.top_p ?? null;
        topK = agent.top_k ?? null;
        minP = agent.min_p ?? null;
        presencePenalty = agent.presence_penalty ?? null;
        maxTokens = agent.max_tokens ?? null;
        enabledSkills = new Set(agent.skills || []);
      } catch {
        goto('/');
        return;
      }
    }

    loading = false;
    // Use tick to let $state settle before snapshotting
    await new Promise(r => setTimeout(r, 0));
    takeSnapshot();
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('beforeunload', handleBeforeunload);
    }
  });

  async function save() {
    if (!name.trim()) {
      error = 'Name is required';
      return;
    }

    const agentSlug = isNew ? slugify(name) : slug;
    if (!agentSlug) {
      error = 'Could not generate a valid slug from name';
      return;
    }

    saving = true;
    error = '';

    try {
      const res = await authenticatedFetch(`/api/agents/${agentSlug}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          slug: agentSlug,
          name: name.trim(),
          tagline: tagline.trim(),
          avatar_emoji: avatarEmoji.trim(),
          system_prompt: systemPrompt.trim(),
          constraints: constraints.trim(),
          greeting: greeting.trim(),
          default_model: defaultModel || undefined,
          temperature: temperature,
          think: thinkEnabled,
          top_p: topP,
          top_k: topK,
          min_p: minP,
          presence_penalty: presencePenalty,
          max_tokens: maxTokens,
          skills: [...enabledSkills],
        })
      });

      if (!res.ok) {
        const text = await res.text();
        error = text || `Error: ${res.status}`;
        return;
      }

      goto(returnTo);
    } catch (err) {
      error = err.message;
    } finally {
      saving = false;
    }
  }

  async function deleteAgent() {
    if (!confirmDelete) {
      confirmDelete = true;
      return;
    }

    saving = true;
    try {
      await authenticatedFetch(`/api/agents/${slug}`, { method: 'DELETE' });
      goto('/');
    } catch (err) {
      error = err.message;
      saving = false;
    }
  }
</script>

{#if loading}
  <div class="editor-container"><p class="loading">Loading...</p></div>
{:else}
<div class="editor-container">
  <Header backHref="/" title={isNew ? 'New Agent' : `Edit ${name}`} />

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <form onsubmit={(e) => { e.preventDefault(); save(); }}>
    <nav class="tabs">
      <button type="button" class="tab" class:active={activeTab === 'persona'}
        onclick={() => activeTab = 'persona'}>Persona</button>
      <button type="button" class="tab" class:active={activeTab === 'conversation'}
        onclick={() => activeTab = 'conversation'}>Conversation</button>
      <button type="button" class="tab" class:active={activeTab === 'sampling'}
        onclick={() => activeTab = 'sampling'}>Sampling</button>
    </nav>

    {#if activeTab === 'persona'}
    <section class="form-section">
      <h2 class="section-heading">Basics</h2>
      <div class="field">
        <label for="name">Name {#if !isNew && slug}<span class="slug-label">/{slug}</span>{/if}</label>
        <input id="name" type="text" bind:value={name} placeholder="My Agent" required />
      </div>
      <div class="field-row">
        <div class="field">
          <label for="emoji">Emoji</label>
          <input id="emoji" type="text" bind:value={avatarEmoji} placeholder="" class="short-input" />
        </div>
        <div class="field flex-1">
          <label for="tagline">Tagline</label>
          <input id="tagline" type="text" bind:value={tagline} placeholder="Reflective companion for deliberate thinking" />
        </div>
      </div>

      <h2 class="section-heading">System Prompt</h2>
      <div class="field">
        <label for="system-prompt">System Prompt</label>
        <textarea id="system-prompt" bind:value={systemPrompt} use:autogrow
          placeholder="Freeform system prompt — define identity, personality, appearance, voice, setting, or anything else"></textarea>
      </div>
      <div class="field">
        <label for="constraints">Constraints</label>
        <textarea id="constraints" bind:value={constraints} use:autogrow
          placeholder="Behavioural boundaries, established ground rules"></textarea>
      </div>

      <button type="button" class="preview-toggle" onclick={() => showPreview = !showPreview}>
        {showPreview ? 'Hide' : 'Show'} Full Prompt Preview
      </button>
      {#if showPreview}
        <pre class="prompt-preview">{buildPreview() || '(empty)'}</pre>
      {/if}
    </section>
    {/if}

    {#if activeTab === 'conversation'}
    <section class="form-section">
      <div class="field">
        <label for="greeting">Greeting</label>
        <textarea id="greeting" bind:value={greeting} use:autogrow
          placeholder="The first thing you say when a conversation opens"></textarea>
      </div>
      <div class="field">
        <label for="model">Default Model</label>
        <select id="model" bind:value={defaultModel}>
          <option value="">None</option>
          {#each models as model}
            <option value={model.name}>{model.name}</option>
          {/each}
        </select>
      </div>
      <div class="field">
        <label>Think</label>
        <button type="button" class="think-toggle" class:active={thinkEnabled}
          onclick={() => thinkEnabled = !thinkEnabled}>
          {thinkEnabled ? 'On' : 'Off'}
        </button>
      </div>
      <div class="field">
        <label>Skills</label>
        {#each availableSkills as skill}
          <label class="checkbox-label">
            <input type="checkbox" checked={enabledSkills.has(skill.id)}
              onchange={() => {
                if (enabledSkills.has(skill.id)) {
                  enabledSkills.delete(skill.id);
                } else {
                  enabledSkills.add(skill.id);
                }
                enabledSkills = new Set(enabledSkills);
              }} />
            {skill.label} — {skill.description}
          </label>
        {/each}
      </div>
    </section>
    {/if}

    {#if activeTab === 'sampling'}
    <section class="form-section">
      <div class="field">
        <label for="temp">Temperature <span class="param-value">{temperature.toFixed(1)}</span></label>
        <input type="range" id="temp" min="0" max="2" step="0.1" bind:value={temperature} />
      </div>
      <div class="field">
        <label for="top-p">Top P {topP != null ? topP.toFixed(2) : '(default)'}</label>
        <input type="range" id="top-p" min="0" max="1" step="0.05"
          value={topP ?? 0.9}
          oninput={(e) => topP = parseFloat(e.target.value)} />
        {#if topP != null}
          <button type="button" class="reset-btn" onclick={() => topP = null}>Reset</button>
        {/if}
      </div>
      <div class="field">
        <label for="top-k">Top K</label>
        <input type="number" id="top-k" min="0" step="1"
          value={topK ?? ''}
          placeholder="Default"
          oninput={(e) => topK = e.target.value ? parseInt(e.target.value) : null} />
      </div>
      <div class="field">
        <label for="min-p">Min P {minP != null ? minP.toFixed(2) : '(default)'}</label>
        <input type="range" id="min-p" min="0" max="1" step="0.05"
          value={minP ?? 0}
          oninput={(e) => minP = parseFloat(e.target.value)} />
        {#if minP != null}
          <button type="button" class="reset-btn" onclick={() => minP = null}>Reset</button>
        {/if}
      </div>
      <div class="field">
        <label for="presence-penalty">Presence Penalty {presencePenalty != null ? presencePenalty.toFixed(1) : '(default)'}</label>
        <input type="range" id="presence-penalty" min="0" max="2" step="0.1"
          value={presencePenalty ?? 0}
          oninput={(e) => presencePenalty = parseFloat(e.target.value)} />
        {#if presencePenalty != null}
          <button type="button" class="reset-btn" onclick={() => presencePenalty = null}>Reset</button>
        {/if}
      </div>
      <div class="field">
        <label for="max-tokens">Max Tokens</label>
        <input type="number" id="max-tokens" min="0" step="1"
          value={maxTokens ?? ''}
          placeholder="Default"
          oninput={(e) => maxTokens = e.target.value ? parseInt(e.target.value) : null} />
      </div>
    </section>
    {/if}

    <footer>
      {#if !isNew}
      <button type="button" class="delete-btn" onclick={deleteAgent} disabled={saving}>
        {confirmDelete ? 'Confirm Delete' : 'Delete'}
      </button>
      {/if}
      <div class="footer-right">
        <a href={returnTo} class="cancel-link">Cancel</a>
        <button type="submit" class="save-btn" class:dirty disabled={saving || !name.trim()}>
          {saving ? 'Saving...' : dirty ? 'Save *' : 'Save'}
        </button>
      </div>
    </footer>
  </form>
</div>
{/if}

<style>
  .editor-container {
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

  .error-banner {
    background: #4a1c2a;
    border: 1px solid #8b3a4a;
    color: #f0a0a0;
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 13px;
    margin-bottom: 16px;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .section-heading {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
    margin-top: 8px;
    padding-bottom: 4px;
    border-bottom: 1px solid var(--border);
  }

  .section-heading:first-child {
    margin-top: 0;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-row {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .flex-1 {
    flex: 1;
  }

  label {
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .slug-label {
    font-weight: 400;
    color: var(--text-muted);
    font-size: 12px;
    margin-left: 6px;
  }

  input[type="text"], select, textarea {
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 8px 10px;
    border-radius: 6px;
    font-family: var(--font-sans);
    font-size: 14px;
    outline: none;
  }

  input[type="text"]:focus, select:focus, textarea:focus {
    border-color: var(--accent);
  }

  input:disabled {
    opacity: 0.5;
  }

  .short-input {
    width: 60px;
  }

  textarea {
    overflow: hidden;
    min-height: 60px;
    line-height: 1.5;
    resize: none;
  }

  select {
    cursor: pointer;
    -webkit-appearance: none;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='8' height='5' viewBox='0 0 8 5'%3E%3Cpath fill='%23888' d='M0 0l4 5 4-5z'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 8px center;
    padding-right: 22px;
  }

  input[type="range"] {
    width: 100%;
    accent-color: var(--accent);
  }

  .tabs {
    display: flex;
    gap: 0;
    border-bottom: 1px solid var(--border);
    margin-bottom: 20px;
  }

  .tab {
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--text-muted);
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
  }

  .tab.active {
    color: var(--text-primary);
    border-bottom-color: var(--accent);
  }

  .tab:hover:not(.active) {
    color: var(--text-secondary);
  }

  .param-value {
    color: var(--text-muted);
    font-weight: 400;
  }

  .reset-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
    align-self: flex-start;
  }

  .reset-btn:hover {
    color: var(--text-secondary);
    border-color: var(--text-secondary);
  }

  input[type="number"] {
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 8px 10px;
    border-radius: 6px;
    font-family: var(--font-sans);
    font-size: 14px;
    outline: none;
    width: 120px;
  }

  input[type="number"]:focus {
    border-color: var(--accent);
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 400;
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    accent-color: var(--accent);
  }

  .think-toggle {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 14px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
  }

  .think-toggle.active {
    background: var(--accent);
    border-color: var(--accent);
    color: white;
  }

  .preview-toggle {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    align-self: flex-start;
  }

  .preview-toggle:hover {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .prompt-preview {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    color: var(--text-secondary);
    max-height: 300px;
    overflow-y: auto;
  }

  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 0;
    border-top: 1px solid var(--border);
    position: sticky;
    bottom: 0;
    background: var(--bg-primary);
  }

  .footer-right {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-left: auto;
  }

  .cancel-link {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 14px;
  }

  .cancel-link:hover {
    color: var(--text-primary);
  }

  .save-btn {
    background: var(--accent);
    border: none;
    color: white;
    padding: 8px 20px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
  }

  .save-btn.dirty {
    box-shadow: 0 0 0 2px var(--bg-primary), 0 0 0 4px #d4a020;
  }

  .save-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .delete-btn {
    background: transparent;
    border: 1px solid #8b3a4a;
    color: #f0a0a0;
    padding: 8px 16px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
  }

  .delete-btn:hover:not(:disabled) {
    background: #4a1c2a;
  }

  .delete-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  @media (max-width: 480px) {
    .editor-container {
      padding: 16px;
    }

    .field-row {
      flex-direction: column;
    }
  }
</style>
