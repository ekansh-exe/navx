import { apiFetch } from "./client";
import type {
  ExecuteTradeRequest,
  ExecuteTradeResponse,
  QuoteRequest,
  QuoteResponse,
  TradeHistoryResponse,
} from "@/types/api";

// Non-binding preview, no mutation — safe to call on every keystroke
// (debounce at the call site anyway).
export function quoteTrade(req: QuoteRequest) {
  return apiFetch<QuoteResponse>("/api/trades/quote", { method: "POST", body: req, auth: true });
}

export function executeTrade(req: ExecuteTradeRequest) {
  return apiFetch<ExecuteTradeResponse>("/api/trades/execute", { method: "POST", body: req, auth: true });
}

export function getMyTrades(params?: { limit?: number; offset?: number }) {
  return apiFetch<TradeHistoryResponse>("/api/users/me/trades", { query: params, auth: true });
}
