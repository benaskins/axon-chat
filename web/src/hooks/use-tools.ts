import { useQuery } from "@tanstack/react-query";
import type { ToolDef } from "@/lib/types";

export function useTools() {
  return useQuery<ToolDef[]>({
    queryKey: ["tools"],
    queryFn: async () => {
      const resp = await fetch("/api/tools");
      return resp.json();
    },
    staleTime: 60_000,
  });
}
