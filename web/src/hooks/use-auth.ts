import { useQuery } from "@tanstack/react-query";

interface AuthUser {
  user_id: string;
  username: string;
}

export function useAuth() {
  return useQuery<AuthUser>({
    queryKey: ["auth"],
    queryFn: async () => {
      const resp = await fetch("/api/me", { credentials: "include" });
      if (!resp.ok) throw new Error("Not authenticated");
      return resp.json();
    },
    retry: false,
  });
}
