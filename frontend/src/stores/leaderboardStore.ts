import { create } from "zustand";
import type { LeaderboardEntry } from "@/types/api";

export interface RankChange {
  direction: "up" | "down";
  previousRank: number;
}

interface LeaderboardState {
  entries: LeaderboardEntry[];
  rankChanges: Map<string, RankChange>;
  setEntries: (entries: LeaderboardEntry[]) => void;
}

// Populated from GET /api/leaderboard on load, then kept current by the
// leaderboard_update WS message (~60s cadence, matching the server's own
// refresh interval). `rankChanges` (keyed by user_id) drives the ↑/↓
// 200ms slide animation for "animate only rows that changed"
// (DESIGN_SPEC_REFINED.md section 7) — computed from real consecutive
// snapshots, not the `change_from_last_refresh` net-worth field.
export const useLeaderboardStore = create<LeaderboardState>((set, get) => ({
  entries: [],
  rankChanges: new Map(),
  setEntries: (entries) => {
    const prev = get().entries;
    const prevByUser = new Map(prev.map((e) => [e.user_id, e.rank]));
    const changes = new Map<string, RankChange>();
    for (const entry of entries) {
      const prevRank = prevByUser.get(entry.user_id);
      if (prevRank !== undefined && prevRank !== entry.rank) {
        changes.set(entry.user_id, {
          direction: entry.rank < prevRank ? "up" : "down",
          previousRank: prevRank,
        });
      }
    }
    set({ entries, rankChanges: changes });
  },
}));
