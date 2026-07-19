import { useMemo } from "react";
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { formatCurrency } from "@/lib/format";
import { useTickHistory } from "@/ws/priceTickStore";
import type { PriceTick } from "@/types/api";

interface ChartPoint {
  price: number;
  ts: string;
}

function ChartTooltip({ active, payload }: { active?: boolean; payload?: { payload: ChartPoint }[] }) {
  if (!active || !payload?.length) return null;
  const point = payload[0].payload;
  return (
    <div className="rounded-button border border-border bg-surface-elevated px-3 py-2 text-xs shadow-hover">
      <div className="font-mono text-sm text-text">{formatCurrency(point.price)}</div>
      <div className="text-text-muted">{new Date(point.ts).toLocaleString()}</div>
    </div>
  );
}

// DESIGN_SPEC_REFINED.md section 6: 520px chart height. GET
// /api/cards/{id}/price-history is proposed/NOT YET IMPLEMENTED
// (API_ENDPOINTS.md), so `restTicks` will usually be empty today — this
// falls back to the session's own live WS ticks, and merges both once the
// backend ships real history (deduped/sorted by timestamp).
export function PriceChart({
  cardId,
  restTicks,
  isLoading,
  isError,
}: {
  cardId: string;
  restTicks: PriceTick[] | undefined;
  isLoading: boolean;
  isError: boolean;
}) {
  const liveTicks = useTickHistory(cardId);

  const data = useMemo<ChartPoint[]>(() => {
    const fromRest: ChartPoint[] = (restTicks ?? []).map((t) => ({ price: t.price, ts: t.ts }));
    const seen = new Set(fromRest.map((p) => p.ts));
    const fromLive = liveTicks.filter((p) => !seen.has(p.ts));
    return [...fromRest, ...fromLive].sort((a, b) => a.ts.localeCompare(b.ts));
  }, [restTicks, liveTicks]);

  if (isLoading) {
    return <div className="h-[520px] animate-pulse rounded-card bg-surface-elevated" />;
  }

  if (isError && data.length === 0) {
    return (
      <div className="flex h-[520px] flex-col items-center justify-center gap-2 rounded-card bg-surface-elevated text-text-muted">
        <span>Chart data unavailable right now.</span>
        <span className="text-xs text-text-disabled">Live prices will appear here as trades happen.</span>
      </div>
    );
  }

  if (data.length < 2) {
    return (
      <div className="flex h-[520px] flex-col items-center justify-center gap-2 rounded-card bg-surface-elevated text-text-muted">
        <span>Not enough price history yet.</span>
        <span className="text-xs text-text-disabled">The chart fills in as live trades come through.</span>
      </div>
    );
  }

  const isUp = data[data.length - 1].price >= data[0].price;
  const stroke = isUp ? "var(--success)" : "var(--danger)";

  return (
    <div className="h-[520px] rounded-card bg-surface-elevated p-4">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 8, left: 8 }}>
          <defs>
            <linearGradient id="priceFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={stroke} stopOpacity={0.25} />
              <stop offset="100%" stopColor={stroke} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="ts"
            tickFormatter={(ts: string) => new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
            stroke="var(--text-disabled)"
            fontSize={11}
            tickLine={false}
            axisLine={false}
            minTickGap={40}
          />
          <YAxis
            domain={["dataMin", "dataMax"]}
            stroke="var(--text-disabled)"
            fontSize={11}
            tickLine={false}
            axisLine={false}
            tickFormatter={(v: number) => formatCurrency(v)}
            width={70}
          />
          <Tooltip content={<ChartTooltip />} />
          <Area
            type="monotone"
            dataKey="price"
            stroke={stroke}
            strokeWidth={2}
            fill="url(#priceFill)"
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
