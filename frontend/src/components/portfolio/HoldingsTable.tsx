import { useNavigate } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency, formatPercent, formatSignedCurrency, formatShares } from "@/lib/format";
import type { Card, Holding } from "@/types/api";

interface Row {
  holding: Holding;
  card: Card | undefined;
}

// Joins the proposed holdings endpoint against the (also proposed) cards
// list client-side — cardDTO is the only place symbol/name/current_price
// live, since HoldingDTO only carries card_id.
export function HoldingsTable({ rows, isLoading }: { rows: Row[]; isLoading: boolean }) {
  const navigate = useNavigate();

  return (
    <div className="overflow-hidden rounded-card border border-border">
      <Table>
        <TableHeader>
          <TableRow className="h-10">
            <TableHead>Symbol</TableHead>
            <TableHead className="text-right">Shares</TableHead>
            <TableHead className="text-right">Avg. cost</TableHead>
            <TableHead className="text-right">Price</TableHead>
            <TableHead className="text-right">Value</TableHead>
            <TableHead className="text-right">P/L</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading &&
            Array.from({ length: 4 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell colSpan={6}>
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              </TableRow>
            ))}

          {!isLoading && rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} className="py-10 text-center text-text-muted">
                No positions yet — trades you make will show up here.
              </TableCell>
            </TableRow>
          )}

          {rows.map(({ holding, card }) => {
            if (!card) return null;
            const value = holding.shares_owned * card.current_price;
            const costBasis = holding.shares_owned * holding.avg_cost_basis;
            const pnl = value - costBasis;
            const pnlPercent = costBasis > 0 ? (pnl / costBasis) * 100 : 0;

            return (
              <TableRow
                key={holding.card_id}
                className="cursor-pointer"
                onClick={() => navigate(`/cards/${card.id}`)}
              >
                <TableCell>
                  <div className="font-medium text-text">{card.symbol}</div>
                  <div className="text-xs text-text-muted">{card.name}</div>
                </TableCell>
                <TableCell className="text-right font-mono">{formatShares(holding.shares_owned)}</TableCell>
                <TableCell className="text-right font-mono">{formatCurrency(holding.avg_cost_basis)}</TableCell>
                <TableCell className="text-right font-mono">{formatCurrency(card.current_price)}</TableCell>
                <TableCell className="text-right font-mono">{formatCurrency(value)}</TableCell>
                <TableCell className={`text-right font-mono ${pnl >= 0 ? "text-success" : "text-danger"}`}>
                  {formatSignedCurrency(pnl)} ({formatPercent(pnlPercent)})
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
