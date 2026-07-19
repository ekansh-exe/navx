import { useQuery } from "@tanstack/react-query";
import { listCards } from "@/api/cards";

// GET /api/cards is marked NOT YET IMPLEMENTED in API_ENDPOINTS.md — this
// will 404 until the backend ships it. staleTime/retry are tuned in
// lib/queryClient.ts to not hammer a route that doesn't exist yet.
export function useCards() {
  return useQuery({
    queryKey: ["cards"],
    queryFn: () => listCards({ limit: 100 }),
  });
}
