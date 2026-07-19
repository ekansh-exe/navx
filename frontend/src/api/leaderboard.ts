import { apiFetch } from "./client";
import type { LeaderboardResponse } from "@/types/api";

// Public, unauthenticated. Server-cached, refreshed ~every 60s — never live.
export function getLeaderboard() {
  return apiFetch<LeaderboardResponse>("/api/leaderboard");
}
