import { useSyncExternalStore } from "react";
import type { PriceTickMessage } from "@/types/ws";

type Tick = PriceTickMessage["data"];
interface HistoryPoint {
  price: number;
  ts: string;
}

const MAX_HISTORY = 60;

// Vanilla (non-React) store for high-frequency price ticks. API_ENDPOINTS.md
// warns to expect very high message volume across all cards combined, so
// updates bypass React state entirely — only components that call
// usePriceTick(cardId)/useTickHistory(cardId) for a currently-visible card
// re-render, and only for that one card.
const ticksByCard = new Map<string, Tick>();
const historyByCard = new Map<string, HistoryPoint[]>();
const volumeByCard = new Map<string, number>();
const listenersByCard = new Map<string, Set<() => void>>();

function notify(cardId: string) {
  const listeners = listenersByCard.get(cardId);
  if (listeners) for (const cb of listeners) cb();
}

function pushHistory(cardId: string, point: HistoryPoint) {
  const existing = historyByCard.get(cardId) ?? [];
  const next = [...existing, point].slice(-MAX_HISTORY);
  historyByCard.set(cardId, next);
}

export function publishTick(tick: Tick) {
  ticksByCard.set(tick.card_id, tick);
  pushHistory(tick.card_id, { price: tick.price, ts: tick.ts });
  volumeByCard.set(tick.card_id, (volumeByCard.get(tick.card_id) ?? 0) + tick.volume);
  notify(tick.card_id);
}

// There's no live price feed or price-history endpoint today (both marked
// NOT YET IMPLEMENTED in API_ENDPOINTS.md), so the session's own tick
// history is the only real data available for a sparkline/daily-change —
// seed it once from GET /api/cards' current_price so there's a baseline the
// moment the card list loads, without clobbering ticks already collected.
export function seedTick(cardId: string, price: number, ts: string) {
  if (historyByCard.has(cardId)) return;
  pushHistory(cardId, { price, ts });
  notify(cardId);
}

function subscribe(cardId: string, onChange: () => void) {
  let listeners = listenersByCard.get(cardId);
  if (!listeners) {
    listeners = new Set();
    listenersByCard.set(cardId, listeners);
  }
  listeners.add(onChange);
  return () => {
    listeners.delete(onChange);
    if (listeners.size === 0) listenersByCard.delete(cardId);
  };
}

/** Latest live price tick for one card, or null until the first tick arrives. */
export function usePriceTick(cardId: string): Tick | null {
  return useSyncExternalStore(
    (onChange) => subscribe(cardId, onChange),
    () => ticksByCard.get(cardId) ?? null
  );
}

const EMPTY_HISTORY: HistoryPoint[] = [];

/** Rolling in-session price history for one card, oldest → newest. */
export function useTickHistory(cardId: string): HistoryPoint[] {
  return useSyncExternalStore(
    (onChange) => subscribe(cardId, onChange),
    () => historyByCard.get(cardId) ?? EMPTY_HISTORY
  );
}

/**
 * Change since the earliest point observed this session — the closest
 * honest proxy for "daily %" available without a real price-history feed.
 * Returns null (render as "—") until at least two distinct points exist.
 */
export function useSessionChange(cardId: string): number | null {
  const history = useTickHistory(cardId);
  if (history.length < 2) return null;
  const first = history[0].price;
  const last = history[history.length - 1].price;
  if (first === 0) return null;
  return ((last - first) / first) * 100;
}

/**
 * Shares traded this session — there's no 24h-volume field on cardDTO and no
 * live feed today, so this is a running sum of WS tick volumes since the tab
 * opened (0 until the first tick, honestly, rather than a fabricated number).
 */
export function useSessionVolume(cardId: string): number {
  return useSyncExternalStore(
    (onChange) => subscribe(cardId, onChange),
    () => volumeByCard.get(cardId) ?? 0
  );
}
