import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { getLeaderboard } from "@/api/leaderboard";
import { useLeaderboardStore } from "@/stores/leaderboardStore";

// GET /api/leaderboard is live today. It's server-cached and refreshed
// ~every 60s (API_ENDPOINTS.md) — polling on the same cadence, plus the WS
// leaderboard_update channel (proposed) pushes into this same query key.
export function useLeaderboard() {
  const setEntries = useLeaderboardStore((s) => s.setEntries);
  const query = useQuery({
    queryKey: ["leaderboard"],
    queryFn: getLeaderboard,
    refetchInterval: 60_000,
  });

  useEffect(() => {
    if (query.data) setEntries(query.data.leaderboard);
  }, [query.data, setEntries]);

  return query;
}
