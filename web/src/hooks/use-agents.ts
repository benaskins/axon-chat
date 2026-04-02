import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authenticatedFetch } from "@/lib/api";
import type { AgentSummary, AgentDetailResponse, Agent } from "@/lib/types";

export function useAgents() {
  return useQuery<AgentSummary[]>({
    queryKey: ["agents"],
    queryFn: async () => {
      const resp = await authenticatedFetch("/api/agents");
      return resp.json();
    },
  });
}

export function useAgent(slug: string | undefined) {
  return useQuery<AgentDetailResponse>({
    queryKey: ["agent", slug],
    queryFn: async () => {
      const resp = await authenticatedFetch(`/api/agents/${slug}`);
      return resp.json();
    },
    enabled: !!slug,
  });
}

export function useSaveAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (agent: Partial<Agent> & { slug: string }) => {
      const resp = await authenticatedFetch(`/api/agents/${agent.slug}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(agent),
      });
      return resp.json();
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
      queryClient.invalidateQueries({ queryKey: ["agent", variables.slug] });
    },
  });
}

export function useDeleteAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (slug: string) => {
      await authenticatedFetch(`/api/agents/${slug}`, { method: "DELETE" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });
}
