<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { authenticatedFetch } from '$lib/auth';
  import { setMenuItems, clearMenuItems } from '$lib/stores/menu.js';
  import Header from '$lib/components/Header.svelte';
  import Markdown from '$lib/components/Markdown.svelte';
  import TaskDrawer from '$lib/components/TaskDrawer.svelte';

  let agent = $state(null);
  let systemPrompt = $state('');
  let messages = $state([]);
  let input = $state('');
  let isStreaming = $state(false);
  let isThinking = $state(false);
  let currentDuration = $state(null);
  let messagesContainer;
  let textareaEl;
  let models = $state([]);
  let selectedModel = $state('');
  let thinkEnabled = $state(true);
  let abortController = $state(null);
  let eventSource = $state(null);
  let connectionStatus = $state('connected'); // 'connected' | 'reconnecting' | 'disconnected'
  let reconnectTimer = null;
  let topicPlaceholder = $state('Type a message...');
  let topicInterval;
  let drawerOpen = $state(false);
  let drawerRef = $state(null);
  let activeTasks = $state(0);

  const topics = [
    'something weird that happened today',
    'what\'s on your mind',
    'the last thing that annoyed you',
    'a song stuck in your head',
    'the hill you\'d die on that nobody cares about',
    'something you saw that stuck with you',
    'what you had for dinner',
    'a rabbit hole you fell down recently',
    'the worst advice you\'ve ever received',
    'something you keep meaning to do',
    'a take you\'re not brave enough to post',
    'the last thing you changed your mind about',
    'something you noticed on your walk',
    'a skill that looks easy but isn\'t',
    'what you\'d build if money didn\'t matter',
    'the strangest compliment you\'ve received',
    'something you know too much about',
    'a place you keep going back to',
    'the last thing that made you laugh',
    'something you believed as a kid',
    'what\'s been keeping you up',
    'a question you don\'t have an answer to',
    'the most overrated thing everyone loves',
    'something you\'re quietly working on',
    'a memory that came back out of nowhere',
  ];

  function updateMenuItems() {
    if (!agent) return;
    const slug = $page.params.slug;
    setMenuItems([
      {
        type: 'select',
        label: 'Model',
        value: selectedModel,
        options: [{ value: '', label: 'Default' }, ...models.map(m => ({ value: m.name, label: m.name }))],
        onchange: (v) => { selectedModel = v; },
        disabled: isStreaming,
      },
      {
        type: 'toggle',
        label: 'Think',
        active: thinkEnabled,
        onclick: () => { thinkEnabled = !thinkEnabled; },
        disabled: isStreaming,
      },
      {
        type: 'button',
        label: 'New Conversation',
        onclick: newChat,
        disabled: isStreaming,
      },
      { type: 'link', label: 'Edit Agent', href: `/agents/${slug}/edit?from=/chat/${slug}/${$page.params.id}` },
    ]);
  }

  // Reactive: update menu items when relevant state changes
  $effect(() => {
    // Touch reactive deps
    void selectedModel;
    void thinkEnabled;
    void isStreaming;
    void models;
    void agent;
    updateMenuItems();
  });

  function connectEventSource() {
    if (eventSource) {
      eventSource.close();
    }

    eventSource = new EventSource('/api/events');
    connectionStatus = 'connected';

    eventSource.onopen = () => {
      if (connectionStatus === 'reconnecting') {
        // Re-fetch conversation to catch any events missed during disconnect
        refreshConversation();
      }
      connectionStatus = 'connected';
    };

    eventSource.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);

        if (data.type === 'image' && data.image) {
          drawerRef?.addImage(data.image);
        }

        if (data.type === 'task' && data.task) {
          drawerRef?.updateTask(data.task);
          // Update active task count
          if (data.task.status === 'completed' || data.task.status === 'failed') {
            activeTasks = Math.max(0, activeTasks - 1);
          }
        }
      } catch {}
    };

    eventSource.onerror = () => {
      connectionStatus = 'reconnecting';
      // Browser auto-reconnects EventSource, but if it stays closed we retry manually
      if (eventSource.readyState === EventSource.CLOSED) {
        connectionStatus = 'disconnected';
        scheduleReconnect();
      }
    };
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connectEventSource();
    }, 3000);
  }

  async function refreshConversation() {
    try {
      const conversationId = $page.params.id;
      const res = await authenticatedFetch(`/api/conversations/${conversationId}`);
      if (!res.ok) return;
      const convo = await res.json();
      if (!convo.messages || convo.messages.length === 0) return;

      const updated = convo.messages.map(m => {
        const existing = messages.find(em => em.id === m.id);
        if (existing) return existing;
        return {
          role: m.role,
          content: m.content,
          thinking: m.thinking || '',
        };
      });
      messages = updated;
    } catch {}
  }

  onMount(async () => {
    const slug = $page.params.slug;
    const conversationId = $page.params.id;

    try {
      const [agentRes, convoRes, modelsRes] = await Promise.all([
        authenticatedFetch(`/api/agents/${slug}`),
        authenticatedFetch(`/api/conversations/${conversationId}`),
        authenticatedFetch('/api/models')
      ]);

      if (!agentRes.ok) {
        goto('/');
        return;
      }

      agent = await agentRes.json();
      systemPrompt = agent.system_prompt;

      if (agent.default_model) {
        selectedModel = agent.default_model;
      }
      thinkEnabled = agent.think ?? true;

      // Load messages from conversation or show greeting
      if (convoRes.ok) {
        const convo = await convoRes.json();
        if (convo.messages && convo.messages.length > 0) {
          messages = convo.messages.map(m => ({
            role: m.role,
            content: m.content,
            thinking: m.thinking || ''
          }));
        } else {
          messages = [{ role: 'assistant', content: agent.greeting }];
        }
      } else {
        messages = [{ role: 'assistant', content: agent.greeting }];
      }

      if (modelsRes.ok) {
        models = await modelsRes.json();
      }

      // Focus textarea after agent loads
      setTimeout(() => {
        scrollToBottom();
        textareaEl?.focus();
      }, 0);
    } catch {
      goto('/');
    }

    // Connect to events stream for async completions (images, tasks)
    connectEventSource();

    // Rotate topic placeholder
    topicPlaceholder = 'Talk about ' + topics[Math.floor(Math.random() * topics.length)] + '...';
    topicInterval = setInterval(() => {
      topicPlaceholder = 'Talk about ' + topics[Math.floor(Math.random() * topics.length)] + '...';
    }, 30000);

    document.addEventListener('keydown', handleGlobalKeydown);
  });

  onDestroy(() => {
    eventSource?.close();
    if (reconnectTimer) clearTimeout(reconnectTimer);
    clearInterval(topicInterval);
    clearMenuItems();
    document.removeEventListener('keydown', handleGlobalKeydown);
  });

  function scrollToBottom() {
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  }

  async function newChat() {
    try {
      const res = await authenticatedFetch(`/api/agents/${$page.params.slug}/conversations`, {
        method: 'POST'
      });
      if (res.ok) {
        const conv = await res.json();
        goto(`/chat/${$page.params.slug}/${conv.id}`);
      }
    } catch {}
  }

  function buildMessagesPayload() {
    // Start with system prompt, then all messages except the placeholder
    const payload = [];
    if (systemPrompt) {
      payload.push({ role: 'system', content: systemPrompt });
    }
    for (const m of messages.slice(0, -1)) {
      payload.push({ role: m.role, content: m.content });
    }
    return payload;
  }

  async function sendMessage() {
    const text = input.trim();
    if (!text || isStreaming) return;

    input = '';
    if (textareaEl) textareaEl.style.height = 'auto';
    messages = [...messages, { role: 'user', content: text }];
    messages = [...messages, { role: 'assistant', content: '', thinking: '' }];
    isStreaming = true;
    isThinking = false;
    currentDuration = null;

    setTimeout(scrollToBottom, 0);

    abortController = new AbortController();

    try {
      const response = await authenticatedFetch('/api/chat', {
        method: 'POST',
        signal: abortController.signal,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          conversation_id: $page.params.id,
          agent_slug: $page.params.slug,
          model: selectedModel || undefined,
          think: thinkEnabled,
          options: Object.fromEntries(
            Object.entries({
              temperature: agent?.temperature ?? 0.7,
              top_p: agent?.top_p,
              top_k: agent?.top_k,
              min_p: agent?.min_p,
              presence_penalty: agent?.presence_penalty,
              num_predict: agent?.max_tokens,
            }).filter(([, v]) => v != null)
          ),
          messages: buildMessagesPayload(),
          tools: agent?.tools || []
        })
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split('\n');
        buffer = lines.pop();

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;

          try {
            const data = JSON.parse(line.slice(6));
            const msg = messages[messages.length - 1];

            if (data.thinking) {
              isThinking = true;
              msg.thinking += data.thinking;
              messages = messages;
              setTimeout(scrollToBottom, 0);
            }

            if (data.content) {
              isThinking = false;
              msg.content += data.content;
              messages = messages;
              setTimeout(scrollToBottom, 0);
            }

            if (data.task) {
              activeTasks++;
              drawerRef?.updateTask({
                task_id: data.task.task_id,
                type: data.task.type,
                status: data.task.status,
                description: data.task.description,
                created_at: new Date().toISOString(),
              });
            }

            if (data.done && data.total_duration_ms !== undefined) {
              currentDuration = data.total_duration_ms;
            }
          } catch {
            // skip malformed SSE lines
          }
        }
      }
    } catch (err) {
      if (err.name !== 'AbortError') {
        messages[messages.length - 1].content = `Error: ${err.message}`;
        messages = messages;
      }
    } finally {
      isStreaming = false;
      isThinking = false;
      abortController = null;
      setTimeout(() => {
        scrollToBottom();
        textareaEl?.focus();
      }, 0);
    }
  }

  function handleKeydown(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      sendMessage();
    }
  }

  function autoGrow() {
    if (!textareaEl) return;
    textareaEl.style.height = 'auto';
    textareaEl.style.height = textareaEl.scrollHeight + 'px';
  }

  function handleGlobalKeydown(e) {
    // Escape: close drawer, abort streaming, or navigate back
    if (e.key === 'Escape' && drawerOpen) {
      drawerOpen = false;
      return;
    }
    if (e.key === 'Escape') {
      if (isStreaming) {
        abortController?.abort();
        return;
      }
      if (document.activeElement === textareaEl && !input.trim()) {
        goto(`/agents/${$page.params.slug}/conversations`);
        return;
      }
      return;
    }

    const mod = e.metaKey || e.ctrlKey;

    // Cmd/Ctrl+Shift+N: New Chat
    if (mod && e.shiftKey && (e.key === 'N' || e.key === 'n')) {
      e.preventDefault();
      if (!isStreaming) newChat();
      return;
    }

    // Cmd/Ctrl+Shift+T: Toggle Think
    if (mod && e.shiftKey && (e.key === 'T' || e.key === 't')) {
      e.preventDefault();
      if (!isStreaming) thinkEnabled = !thinkEnabled;
      return;
    }
  }
</script>

{#if agent}
<div class="chat-container">
  <Header backHref="/agents/{$page.params.slug}/conversations" title="{agent.avatar_emoji} {agent.name}" borderBottom>
    {#snippet right()}
      <button class="new-chat" onclick={newChat} disabled={isStreaming}>
        New Conversation
      </button>
      <button class="tasks-btn" onclick={() => drawerOpen = !drawerOpen}>
        Tasks
        {#if activeTasks > 0}
          <span class="tasks-badge">{activeTasks}</span>
        {/if}
      </button>
    {/snippet}
  </Header>

  {#if connectionStatus !== 'connected'}
    <div class="connection-status {connectionStatus}">
      {connectionStatus === 'reconnecting' ? 'Reconnecting...' : 'Connection lost — retrying...'}
    </div>
  {/if}

  <div class="messages" bind:this={messagesContainer}>
    {#each messages as message, i}
      <div class="message {message.role}">
        {#if message.thinking}
          {#if isStreaming && isThinking && i === messages.length - 1}
            <div class="thinking-live">
              <div class="thinking-label">Thinking</div>
              <div class="thinking-content">
                {message.thinking}<span class="cursor">|</span>
              </div>
            </div>
          {:else}
            <details class="thinking-block">
              <summary>Thinking</summary>
              <div class="thinking-content">{message.thinking}</div>
            </details>
          {/if}
        {/if}
        <div class="message-content">
          {#if message.role === 'assistant' && isStreaming && !isThinking && !message.content && i === messages.length - 1}
            <span class="typing-dots"><span></span><span></span><span></span></span>
          {:else if message.role === 'assistant'}
            <Markdown content={message.content} />
            {#if isStreaming && !isThinking && i === messages.length - 1}
              <span class="cursor">|</span>
            {/if}
          {:else}
            {message.content}
          {/if}
        </div>
        {#if message.role === 'assistant' && !isStreaming && i === messages.length - 1 && currentDuration !== null}
          <div class="duration">{currentDuration}ms</div>
        {/if}
      </div>
    {/each}
  </div>

  <div class="input-area">
    <textarea
      bind:this={textareaEl}
      bind:value={input}
      onkeydown={handleKeydown}
      oninput={autoGrow}
      placeholder={topicPlaceholder}
      disabled={isStreaming}
      rows="1"
    ></textarea>
    <button class="send" onclick={sendMessage} disabled={!input.trim() || isStreaming}>
      Send
    </button>
  </div>
</div>
{/if}

<TaskDrawer
  bind:this={drawerRef}
  open={drawerOpen}
  agentSlug={$page.params.slug}
  onclose={() => drawerOpen = false}
/>

<style>
  .connection-status {
    text-align: center;
    padding: 4px 12px;
    font-size: 12px;
    color: var(--text-secondary);
    background: var(--bg-tertiary);
    border-bottom: 1px solid var(--border);
  }
  .connection-status.disconnected {
    color: var(--error, #e74c3c);
  }

  .chat-container {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
  }

  .new-chat {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .new-chat:hover:not(:disabled) {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .new-chat:active:not(:disabled) {
    background: var(--border);
  }

  .new-chat:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 24px;
    -webkit-overflow-scrolling: touch;
  }

  .message {
    max-width: 85%;
    font-size: 15px;
    line-height: 1.7;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }

  .message.user {
    align-self: flex-end;
    background: var(--bg-user-msg);
    padding: 10px 16px;
    border-radius: 12px;
    border-bottom-right-radius: 4px;
    white-space: pre-wrap;
  }

  .message.assistant {
    align-self: flex-start;
    max-width: 100%;
    padding: 4px 0;
  }

  .message-content {
    font-family: var(--font-sans);
  }

  .cursor {
    animation: blink 1s step-end infinite;
    color: var(--accent);
  }

  @keyframes blink {
    50% { opacity: 0; }
  }

  .thinking-live, .thinking-block {
    font-size: 12px;
    color: var(--text-muted);
    border-left: 2px solid var(--accent);
    padding-left: 8px;
    margin-bottom: 8px;
  }

  .thinking-label {
    font-size: 12px;
    color: var(--text-muted);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .thinking-block summary {
    cursor: pointer;
    font-size: 12px;
    color: var(--text-muted);
  }

  .thinking-content {
    white-space: pre-wrap;
    margin-top: 4px;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  .duration {
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  .input-area {
    display: flex;
    gap: 8px;
    padding: 12px 16px;
    border-top: 1px solid var(--border);
    background: var(--bg-secondary);
    flex-shrink: 0;
  }

  textarea {
    flex: 1;
    min-width: 0;
    min-height: 44px;
    max-height: 200px;
    overflow-y: auto;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 12px 16px;
    border-radius: 16px;
    font-family: var(--font-sans);
    font-size: 16px;
    resize: none;
    outline: none;
    line-height: 1.5;
    -webkit-appearance: none;
    transition: border-color 0.15s, box-shadow 0.15s;
  }

  textarea:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(201, 169, 97, 0.15);
  }

  textarea:disabled {
    opacity: 0.5;
  }

  .send {
    background: var(--accent);
    border: none;
    color: white;
    padding: 10px 16px;
    border-radius: 12px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    flex-shrink: 0;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .send:hover:not(:disabled) {
    opacity: 0.9;
  }

  .send:active:not(:disabled) {
    opacity: 0.8;
  }

  .send:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .tasks-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    position: relative;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .tasks-btn:hover {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .tasks-badge {
    position: absolute;
    top: -6px;
    right: -6px;
    background: var(--accent);
    color: white;
    font-size: 10px;
    font-weight: 600;
    min-width: 16px;
    height: 16px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 4px;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .typing-dots {
    display: inline-flex;
    gap: 4px;
    padding: 4px 0;
  }

  .typing-dots span {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-muted);
    animation: typing-fade 1.4s ease-in-out infinite;
  }

  .typing-dots span:nth-child(2) {
    animation-delay: 0.2s;
  }

  .typing-dots span:nth-child(3) {
    animation-delay: 0.4s;
  }

  @keyframes typing-fade {
    0%, 80%, 100% { opacity: 0.2; }
    40% { opacity: 1; }
  }

  @media (max-width: 480px) {
    .message {
      max-width: 90%;
      font-size: 16px;
      padding: 8px 12px;
    }

    textarea {
      font-size: 18px;
    }

    .send {
      padding: 10px 12px;
    }
  }
</style>
