import { useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router";
import { Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AppHeader } from "@/components/app-header";
import { MenuButton } from "@/components/menu-button";
import { MarkdownContent } from "@/components/markdown-content";
import { useMenu } from "@/components/menu-context";
import { useAgent } from "@/hooks/use-agents";
import { useConversation, useCreateConversation } from "@/hooks/use-conversations";
import { useModels } from "@/hooks/use-models";
import type { Message, SSEEvent } from "@/lib/types";
import { authenticatedFetch } from "@/lib/api";

export default function ChatPage() {
  const { slug, id } = useParams();
  const navigate = useNavigate();
  const { setItems, clearItems } = useMenu();
  const { data: agent } = useAgent(slug);
  const { data: conversationData } = useConversation(id);
  const { data: models } = useModels();
  const createConversation = useCreateConversation();

  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [streamingThinking, setStreamingThinking] = useState("");
  const [selectedModel, setSelectedModel] = useState("");
  const [thinkEnabled, setThinkEnabled] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Load messages from conversation data
  useEffect(() => {
    if (conversationData?.Messages) {
      setMessages(conversationData.Messages);
    }
  }, [conversationData]);

  // Set default model from agent
  useEffect(() => {
    if (agent?.default_model && !selectedModel) {
      setSelectedModel(agent.default_model);
    }
    if (agent?.think !== null && agent?.think !== undefined) {
      setThinkEnabled(agent.think);
    }
  }, [agent, selectedModel]);

  // Menu items
  useEffect(() => {
    const items = [];
    if (models?.length) {
      items.push({
        type: "select" as const,
        label: "Model",
        value: selectedModel,
        options: models.map((m) => ({ label: m.Name, value: m.Name })),
        onChange: setSelectedModel,
      });
    }
    items.push({
      type: "toggle" as const,
      label: "Think",
      value: thinkEnabled,
      onToggle: () => setThinkEnabled((v) => !v),
    });
    if (slug) {
      items.push({
        type: "button" as const,
        label: "New Conversation",
        onClick: async () => {
          const conv = await createConversation.mutateAsync(slug);
          navigate(`/chat/${slug}/${conv.id}`);
        },
      });
      items.push({
        type: "link" as const,
        label: "Edit Agent",
        href: `/agents/${slug}/edit?from=/chat/${slug}/${id}`,
      });
    }
    setItems(items);
    return clearItems;
  }, [models, selectedModel, thinkEnabled, slug, id, setItems, clearItems, createConversation, navigate]);

  // Auto-scroll
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamingContent]);

  // Auto-resize textarea
  function handleInputChange(value: string) {
    setInput(value);
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  }

  async function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;

    const userMessage: Message = {
      id: crypto.randomUUID(),
      conversation_id: id || "",
      role: "user",
      content: text,
      thinking: "",
      tool_calls: "",
      images: "",
      duration_ms: null,
      created_at: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    if (textareaRef.current) textareaRef.current.style.height = "auto";
    setIsStreaming(true);
    setStreamingContent("");
    setStreamingThinking("");

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const allMessages = [...messages, userMessage].map((m) => ({
        role: m.role,
        content: m.content,
      }));

      const resp = await authenticatedFetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          conversation_id: id,
          agent_slug: slug,
          model: selectedModel,
          think: thinkEnabled || undefined,
          messages: allMessages,
          tools: agent?.tools || [],
        }),
        signal: controller.signal,
      });

      const reader = resp.body?.getReader();
      if (!reader) return;

      const decoder = new TextDecoder();
      let buffer = "";
      let fullContent = "";
      let fullThinking = "";
      let durationMs: number | null = null;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;
          try {
            const event: SSEEvent = JSON.parse(line.slice(6));
            if (event.content) {
              fullContent += event.content;
              setStreamingContent(fullContent);
            }
            if (event.thinking) {
              fullThinking += event.thinking;
              setStreamingThinking(fullThinking);
            }
            if (event.done) {
              durationMs = event.total_duration_ms ?? null;
            }
          } catch {
            // skip malformed lines
          }
        }
      }

      const assistantMessage: Message = {
        id: crypto.randomUUID(),
        conversation_id: id || "",
        role: "assistant",
        content: fullContent,
        thinking: fullThinking,
        tool_calls: "",
        images: "",
        duration_ms: durationMs,
        created_at: new Date().toISOString(),
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        console.error("Chat error:", err);
      }
    } finally {
      setIsStreaming(false);
      setStreamingContent("");
      setStreamingThinking("");
      abortRef.current = null;
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    if (e.key === "Escape" && isStreaming) {
      abortRef.current?.abort();
    }
  }

  const title = agent ? `${agent.avatar_emoji} ${agent.name}` : "Chat";

  return (
    <div className="flex flex-col h-screen">
      <AppHeader
        backHref={slug ? `/agents/${slug}/conversations` : "/"}
        title={title}
        rightContent={<MenuButton />}
      />

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {agent?.greeting && messages.length === 0 && !isStreaming && (
          <div className="text-muted-foreground italic">{agent.greeting}</div>
        )}

        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-[80%] rounded-lg px-4 py-2 ${
                msg.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted"
              }`}
            >
              {msg.thinking && (
                <details className="mb-2">
                  <summary className="text-xs text-muted-foreground cursor-pointer">
                    Thinking
                  </summary>
                  <div className="mt-1 text-xs text-muted-foreground whitespace-pre-wrap">
                    {msg.thinking}
                  </div>
                </details>
              )}
              {msg.role === "assistant" ? (
                <MarkdownContent content={msg.content} />
              ) : (
                <div className="whitespace-pre-wrap">{msg.content}</div>
              )}
              {msg.duration_ms && (
                <div className="text-xs text-muted-foreground mt-1">
                  {msg.duration_ms}ms
                </div>
              )}
            </div>
          </div>
        ))}

        {isStreaming && (
          <div className="flex justify-start">
            <div className="max-w-[80%] rounded-lg px-4 py-2 bg-muted">
              {streamingThinking && (
                <div className="text-xs text-muted-foreground mb-2">
                  Thinking...
                </div>
              )}
              {streamingContent ? (
                <MarkdownContent content={streamingContent} />
              ) : (
                <span className="inline-flex gap-1">
                  <span className="animate-bounce">.</span>
                  <span className="animate-bounce [animation-delay:0.1s]">.</span>
                  <span className="animate-bounce [animation-delay:0.2s]">.</span>
                </span>
              )}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <div className="border-t p-4">
        <div className="flex gap-2 items-end">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => handleInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            className="flex-1 min-h-[40px] max-h-[200px] resize-none rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            rows={1}
          />
          <Button onClick={handleSend} disabled={!input.trim() || isStreaming}>
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
