import { Card, CardContent } from "@/components/ui/card";
import { useLeaderboard } from "@/hooks/useLeaderboard";
import { useAuthStore } from "@/stores/authStore";
import { LeaderboardTable } from "@/components/leaderboard/LeaderboardTable";

// DESIGN_SPEC_REFINED.md section 6 ("Leaderboard"): top-3 accent strip,
// current-user highlight, rank-change ↑/↓. Only the top 100 by net worth are
// returned and there's no "your rank" lookup (API_ENDPOINTS.md) — a user
// outside the list gets an honest note instead of a fabricated pinned row.
export function LeaderboardPage() {
  const { data, isLoading } = useLeaderboard();
  const user = useAuthStore((s) => s.user);

  const entries = data?.leaderboard ?? [];
  const isCurrentUserListed = entries.some((e) => e.user_id === user?.id);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-text">Leaderboard</h1>

      {user && !isLoading && !isCurrentUserListed && (
        <Card>
          <CardContent className="text-sm text-text-muted">
            You're outside the top 100 — there's no ranking beyond that today.
          </CardContent>
        </Card>
      )}

      <LeaderboardTable entries={entries} isLoading={isLoading} currentUserId={user?.id} />
    </div>
  );
}
