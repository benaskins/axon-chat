export interface AgentSummary {
  slug: string;
  name: string;
  tagline: string;
  avatar_emoji: string;
  default_model: string;
}

export interface Agent {
  slug: string;
  user_id: string;
  name: string;
  tagline: string;
  avatar_emoji: string;
  system_prompt: string;
  constraints: string;
  greeting: string;
  default_model: string;
  temperature: number | null;
  think: boolean | null;
  top_p: number | null;
  top_k: number | null;
  min_p: number | null;
  presence_penalty: number | null;
  max_tokens: number | null;
  tools: string[];
}

export interface AgentDetailResponse extends Agent {
  full_prompt: string;
}

export interface Conversation {
  id: string;
  agent_slug: string;
  user_id: string;
  title: string | null;
  created_at: string;
  updated_at: string;
}

export interface ConversationSummary {
  id: string;
  title: string | null;
  created_at: string;
  updated_at: string;
  message_count: number;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  thinking: string;
  tool_calls: string;
  images: string;
  duration_ms: number | null;
  created_at: string;
}

export interface ConversationWithMessages {
  Conversation: Conversation;
  Messages: Message[];
}

export interface ToolDef {
  name: string;
  description: string;
}

export interface Task {
  id: string;
  type: string;
  status: string;
  description: string;
  result_summary: string;
  error: string;
  artifact_id: string;
  created_at: string;
  updated_at: string;
}

export interface GalleryImage {
  id: string;
  prompt: string;
  model: string;
  url: string;
  is_base: boolean;
  is_nsfw: boolean;
  created_at: string;
}

export interface SSEEvent {
  content?: string;
  thinking?: string;
  done?: boolean;
  total_duration_ms?: number;
  tool_use?: string;
  task?: Task;
}
