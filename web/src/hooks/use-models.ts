import { useQuery } from "@tanstack/react-query";
import { authenticatedFetch } from "@/lib/api";

interface Model {
  Name: string;
}

export function useModels() {
  return useQuery<Model[]>({
    queryKey: ["models"],
    queryFn: async () => {
      const resp = await authenticatedFetch("/api/models");
      if (!resp.ok) return [];
      return resp.json();
    },
    staleTime: 60_000,
  });
}
