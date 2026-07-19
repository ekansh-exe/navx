import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { formatCurrency, formatSignedCurrency } from "@/lib/format";
import { useLeaderboardStore } from "@/stores/leaderboardStore";
import type { LeaderboardEntry } from "@/types/api";

const RANK_ACCENT: Record<number, string> = {
  1: "border-l-4 border-l-gold",
  2: "border-l-4 border-l-silver",
  3: "border-l-4 border-l-bronze",
};

// Real, rank-derived tiers only — "Momentum Master"/"Diamond Hands"/etc. from
// the spec's example chips have no backing data anywhere in the API and
// would be fabricated per-user claims, so they're intentionally omitted.
function rankTierBadge(rank: number, total: number) {
  if (rank <= Math.max(1, Math.ceil(total * 0.01))) return <Badge variant="gold">Top 1%</Badge>;
  if (rank <= Math.max(1, Math.ceil(total * 0.1))) return <Badge variant="info">Top 10%</Badge>;
  return null;
}

export function LeaderboardTable({
  entries,
  isLoading,
  currentUserId,
}: {
  entries: LeaderboardEntry[];
  isLoading: boolean;
  currentUserId: string | undefined;
}) {
  const rankChanges = useLeaderboardStore((s) => s.rankChanges);
  // The GOAT tribute row (see backend leaderboard.GoatEntry) is a fixed
  // decoration, not a ranked user — excluded from the percentile math below
  // so it can't count toward its own "Top 1%" badge.
  const rankedCount = entries.filter((e) => !e.is_goat).length;

  return (
    <div className="overflow-hidden rounded-card border border-border">
      <Table>
        <TableHeader>
          <TableRow className="h-10">
            <TableHead className="w-16">Rank</TableHead>
            <TableHead>Trader</TableHead>
            <TableHead className="text-right">Net worth</TableHead>
            <TableHead className="text-right">Change</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading &&
            Array.from({ length: 8 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell colSpan={4}>
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              </TableRow>
            ))}

          {!isLoading && entries.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} className="py-10 text-center text-text-muted">
                Leaderboard is empty right now.
              </TableCell>
            </TableRow>
          )}

          {entries.map((entry) => {
            if (entry.is_goat) {
              return (
                <TableRow
                  key={entry.user_id}
                  className="border-l-4 border-l-gold bg-gold-glow hover:bg-gold-glow"
                >
                  <TableCell className="font-mono">
                    <Badge variant="gold">GOAT</Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-base font-semibold text-gold">{entry.username}</span>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm font-medium text-gold" colSpan={2}>
                    {entry.net_worth_display}
                  </TableCell>
                </TableRow>
              );
            }

            const isCurrentUser = entry.user_id === currentUserId;
            const rankChange = rankChanges.get(entry.user_id);

            return (
              <TableRow
                key={entry.user_id}
                className={cn(
                  RANK_ACCENT[entry.rank],
                  isCurrentUser && "border-primary bg-primary/10"
                )}
              >
                <TableCell className="font-mono">
                  <span
                    className={cn(
                      "inline-flex items-center gap-1",
                      rankChange && "animate-in fade-in slide-in-from-bottom-1 duration-200"
                    )}
                  >
                    {entry.rank}
                    {rankChange && (
                      <span className={rankChange.direction === "up" ? "text-success" : "text-danger"}>
                        {rankChange.direction === "up" ? "▲" : "▼"}
                      </span>
                    )}
                  </span>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <span className={cn("font-medium", isCurrentUser ? "text-primary" : "text-text")}>
                      {entry.username}
                    </span>
                    {isCurrentUser && <Badge variant="outline">You</Badge>}
                    {rankTierBadge(entry.rank, rankedCount)}
                  </div>
                </TableCell>
                <TableCell className="text-right font-mono">{formatCurrency(entry.net_worth)}</TableCell>
                <TableCell className="text-right font-mono">
                  {entry.change_from_last_refresh === undefined ? (
                    <span className="text-text-muted">—</span>
                  ) : (
                    <span className={entry.change_from_last_refresh >= 0 ? "text-success" : "text-danger"}>
                      {formatSignedCurrency(entry.change_from_last_refresh)}
                    </span>
                  )}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
