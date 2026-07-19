// Proposed shapes from API_ENDPOINTS.md section 7 — internal/ws does not exist
// in the backend yet, so these are speculative but kept schema-compatible
// with the REST DTOs (same field names/casing/units).

import type { LeaderboardEntry, NewsEvent } from "./api";

export interface PriceTickMessage {
  type: "price_tick";
  data: {
    card_id: string;
    symbol: string;
    price: number;
    previous_price: number;
    volume: number;
    ts: string;
  };
}

export interface NewsPublishedMessage {
  type: "news_published";
  data: NewsEvent;
}

export interface LeaderboardUpdateMessage {
  type: "leaderboard_update";
  data: {
    leaderboard: LeaderboardEntry[];
  };
}

export type WsMessage =
  | PriceTickMessage
  | NewsPublishedMessage
  | LeaderboardUpdateMessage;

export type ConnectionStatus = "connecting" | "live" | "disconnected";
