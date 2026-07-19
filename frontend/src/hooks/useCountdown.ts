import { useEffect, useState } from "react";

function format(ms: number): string {
  if (ms <= 0) return "00:00:00";
  const totalSeconds = Math.floor(ms / 1000);
  const h = Math.floor(totalSeconds / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  const s = totalSeconds % 60;
  return [h, m, s].map((n) => String(n).padStart(2, "0")).join(":");
}

/** Live "HH:MM:SS" countdown to an ISO timestamp, ticking every second. */
export function useCountdown(targetIso: string): string {
  const [label, setLabel] = useState(() => format(new Date(targetIso).getTime() - Date.now()));

  useEffect(() => {
    const target = new Date(targetIso).getTime();
    const interval = setInterval(() => setLabel(format(target - Date.now())), 1000);
    return () => clearInterval(interval);
  }, [targetIso]);

  return label;
}
