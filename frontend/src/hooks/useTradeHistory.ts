import { useQuery } from "@tanstack/react-query";
import { getMyTrades } from "@/api/trades";

// Persisted server-side history (GET /api/users/me/trades) — replaces the
// old session-only client store, which lost everything on refresh or a
// fresh login since it was never backed by anything but in-memory state.
export function useTradeHistory() {
  return useQuery({
    queryKey: ["trades"],
    queryFn: () => getMyTrades({ limit: 20 }),
  });
}
