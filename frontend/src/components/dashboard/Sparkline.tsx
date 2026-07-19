import { Line, LineChart, ResponsiveContainer, YAxis } from "recharts";

interface HistoryPoint {
  price: number;
  ts: string;
}

// Bare-axes trend line — company cards get a small one, NAV5 a taller one.
// Section 8/section 7 rules ("never animate prices") apply to the numeric
// readouts, not this chart; recharts' own transition is fine here since it's
// visualizing a trend, not a live-ticking number.
export function Sparkline({ data, height = 32 }: { data: HistoryPoint[]; height?: number }) {
  const isUp = data.length >= 2 ? data[data.length - 1].price >= data[0].price : true;
  const stroke = data.length < 2 ? "var(--text-disabled)" : isUp ? "var(--success)" : "var(--danger)";

  if (data.length === 0) {
    return <div style={{ height }} className="flex items-center text-xs text-text-disabled">No data yet</div>;
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
        <YAxis hide domain={["dataMin", "dataMax"]} />
        <Line
          type="monotone"
          dataKey="price"
          stroke={stroke}
          strokeWidth={1.5}
          dot={false}
          isAnimationActive={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
