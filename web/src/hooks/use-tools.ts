import { useQuery } from "@tanstack/react-query";
import { authenticatedFetch } from "@/lib/api";
import type { ToolDef } from "@/lib/types";

export function useTools() {
  return useQuery<ToolDef[]>({
    queryKey: ["tools"],
    queryFn: async () => {
      const resp = await authenticatedFetch("/api/tools");
      if (!resp.ok) return [];
      return resp.json();
    },
    staleTime: 60_000,
  });
}
