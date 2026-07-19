import { cn } from "@/lib/utils";
import { useConnectionStatus } from "@/ws/WebSocketProvider";

const LABEL: Record<string, string> = {
  live: "LIVE",
  connecting: "CONNECTING",
  disconnected: "DISCONNECTED",
};

const DOT_CLASS: Record<string, string> = {
  live: "bg-success",
  connecting: "bg-text-muted animate-pulse",
  disconnected: "bg-danger",
};

// DESIGN_SPEC_REFINED.md section 7 ("Connection Status"):
// green = LIVE, gray = CONNECTING, red = DISCONNECTED. Header indicator.
export function ConnectionIndicator() {
  const status = useConnectionStatus();

  return (
    <div className="flex items-center gap-1.5 text-xs font-medium text-text-muted">
      <span className={cn("size-1.5 rounded-full", DOT_CLASS[status])} aria-hidden />
      <span className="uppercase tracking-wide">{LABEL[status]}</span>
    </div>
  );
}
