import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { formatCurrency, formatShares } from "@/lib/format";
import { useTradeHistoryStore } from "@/stores/tradeHistoryStore";

// There's no GET-transactions endpoint anywhere (API_ENDPOINTS.md) — this is
// this session's own confirmed trades only, labeled honestly rather than
// implying a full history.
export function RecentTradesTable() {
  const entries = useTradeHistoryStore((s) => s.entries);

  return (
    <div className="overflow-hidden rounded-card border border-border">
      <Table>
        <TableHeader>
          <TableRow className="h-10">
            <TableHead>Trade</TableHead>
            <TableHead className="text-right">Shares</TableHead>
            <TableHead className="text-right">Price</TableHead>
            <TableHead className="text-right">Fee</TableHead>
            <TableHead className="text-right">Total</TableHead>
            <TableHead className="text-right">Time</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} className="py-10 text-center text-text-muted">
                No trades yet this session.
              </TableCell>
            </TableRow>
          )}

          {entries.map(({ transaction, feeTransaction, card }) => (
            <TableRow key={transaction.id}>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Badge variant={transaction.type === "BUY" ? "success" : "danger"}>
                    {transaction.type}
                  </Badge>
                  <span className="font-medium text-text">{card.symbol}</span>
                </div>
              </TableCell>
              <TableCell className="text-right font-mono">
                {formatShares(transaction.shares ?? 0)}
              </TableCell>
              <TableCell className="text-right font-mono">
                {formatCurrency(transaction.price_per_share ?? 0)}
              </TableCell>
              <TableCell className="text-right font-mono">
                {formatCurrency(Math.abs(feeTransaction.total_currency_delta))}
              </TableCell>
              <TableCell className="text-right font-mono">
                {formatCurrency(Math.abs(transaction.total_currency_delta))}
              </TableCell>
              <TableCell className="text-right text-text-muted">
                {new Date(transaction.created_at).toLocaleTimeString()}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
