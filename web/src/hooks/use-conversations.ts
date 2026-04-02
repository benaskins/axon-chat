import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authenticatedFetch } from "@/lib/api";
import type { ConversationSummary, ConversationWithMessages } from "@/lib/types";

export function useConversations(slug: string | undefined) {
  return useQuery<ConversationSummary[]>({
    queryKey: ["conversations", slug],
    queryFn: async () => {
      const resp = await authenticatedFetch(
        `/api/agents/${slug}/conversations`
      );
      return resp.json();
    },
    enabled: !!slug,
  });
}

export function useConversation(id: string | undefined) {
  return useQuery<ConversationWithMessages>({
    queryKey: ["conversation", id],
    queryFn: async () => {
      const resp = await authenticatedFetch(`/api/conversations/${id}`);
      return resp.json();
    },
    enabled: !!id,
  });
}

export function useCreateConversation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (slug: string) => {
      const resp = await authenticatedFetch(
        `/api/agents/${slug}/conversations`,
        { method: "POST" }
      );
      return resp.json();
    },
    onSuccess: (_data, slug) => {
      queryClient.invalidateQueries({ queryKey: ["conversations", slug] });
    },
  });
}

export function useDeleteConversation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await authenticatedFetch(`/api/conversations/${id}`, {
        method: "DELETE",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["conversations"] });
    },
  });
}
