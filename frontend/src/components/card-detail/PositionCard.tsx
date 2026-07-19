import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatCurrency, formatPercent, formatSignedCurrency, formatShares } from "@/lib/format";
import { useHoldingsStore } from "@/stores/holdingsStore";
import type { Card as CardData } from "@/types/api";

// DESIGN_SPEC_REFINED.md section 6: "Your Position" is a separate elevated
// card — shares, avg cost, current value, P/L — never mixed with market
// stats. Backed by the proposed GET /api/users/me/holdings (see
// hooks/useHoldings.ts) — shows an honest empty state until that exists.
export function PositionCard({ card, isLoading }: { card: CardData; isLoading: boolean }) {
  const holding = useHoldingsStore((s) => s.byCardId[card.id]);

  const currentValue = holding ? holding.shares_owned * card.current_price : 0;
  const costBasisValue = holding ? holding.shares_owned * holding.avg_cost_basis : 0;
  const pnl = currentValue - costBasisValue;
  const pnlPercent = costBasisValue > 0 ? (pnl / costBasisValue) * 100 : 0;

  return (
    <Card className="bg-surface-elevated">
      <CardHeader>
        <CardTitle className="text-base">Your Position</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <p className="text-sm text-text-muted">Loading position…</p>
        ) : !holding || holding.shares_owned === 0 ? (
          <p className="text-sm text-text-muted">You don't own any shares of {card.symbol} yet.</p>
        ) : (
          <dl className="grid grid-cols-2 gap-y-3 text-sm">
            <dt className="text-text-muted">Shares</dt>
            <dd className="text-right font-mono text-text">{formatShares(holding.shares_owned)}</dd>

            <dt className="text-text-muted">Avg. cost</dt>
            <dd className="text-right font-mono text-text">{formatCurrency(holding.avg_cost_basis)}</dd>

            <dt className="text-text-muted">Current value</dt>
            <dd className="text-right font-mono text-text">{formatCurrency(currentValue)}</dd>

            <dt className="text-text-muted">P/L</dt>
            <dd className={`text-right font-mono ${pnl >= 0 ? "text-success" : "text-danger"}`}>
              {formatSignedCurrency(pnl)} ({formatPercent(pnlPercent)})
            </dd>
          </dl>
        )}
      </CardContent>
    </Card>
  );
}
