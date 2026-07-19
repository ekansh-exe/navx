import { Card, CardContent } from "@/components/ui/card";
import { useHoldings } from "@/hooks/useHoldings";
import { useCards } from "@/hooks/useCards";
import { useAuthStore } from "@/stores/authStore";
import { HoldingsTable } from "@/components/portfolio/HoldingsTable";
import { RecentTradesTable } from "@/components/portfolio/RecentTradesTable";
import { formatCurrency } from "@/lib/format";

export function PortfolioPage() {
  const user = useAuthStore((s) => s.user);
  const holdingsQuery = useHoldings();
  const cardsQuery = useCards();

  const cardsById = new Map((cardsQuery.data?.cards ?? []).map((c) => [c.id, c]));
  const holdings = holdingsQuery.data?.holdings ?? [];
  const rows = holdings.map((holding) => ({ holding, card: cardsById.get(holding.card_id) }));

  const cashBalance = user?.currency_balance ?? 0;
  const holdingsValue = rows.reduce(
    (sum, { holding, card }) => sum + (card ? holding.shares_owned * card.current_price : 0),
    0
  );
  const netWorth = cashBalance + holdingsValue;

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-text">Portfolio</h1>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
        <Card>
          <CardContent className="flex flex-col gap-1">
            <span className="text-xs text-text-muted uppercase">Net worth</span>
            <span className="font-mono text-2xl font-semibold text-text">{formatCurrency(netWorth)}</span>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex flex-col gap-1">
            <span className="text-xs text-text-muted uppercase">Cash balance</span>
            <span className="font-mono text-2xl font-semibold text-text">{formatCurrency(cashBalance)}</span>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex flex-col gap-1">
            <span className="text-xs text-text-muted uppercase">Holdings value</span>
            <span className="font-mono text-2xl font-semibold text-text">{formatCurrency(holdingsValue)}</span>
          </CardContent>
        </Card>
      </div>

      <div className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold text-text">Positions</h2>
        <HoldingsTable rows={rows} isLoading={holdingsQuery.isLoading || cardsQuery.isLoading} />
      </div>

      <div className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold text-text">Recent trades (this session)</h2>
        <RecentTradesTable />
      </div>
    </div>
  );
}
