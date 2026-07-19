import { cn } from "@/lib/utils";
import { formatPercent } from "@/lib/format";

// DESIGN_SPEC_REFINED.md section 6 ("Price Movement"): use ▲/▼, not color
// alone, for accessibility. `percent` is null when no baseline exists yet
// (see ws/priceTickStore's useSessionChange) — render a neutral dash rather
// than a fabricated 0%.
export function PriceChangeIndicator({ percent, className }: { percent: number | null; className?: string }) {
  if (percent === null) {
    return <span className={cn("font-mono text-[13px] font-medium text-text-muted", className)}>—</span>;
  }

  const isUp = percent >= 0;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 font-mono text-[13px] font-medium",
        isUp ? "text-success" : "text-danger",
        className
      )}
    >
      <span aria-hidden>{isUp ? "▲" : "▼"}</span>
      {formatPercent(percent)}
    </span>
  );
}
