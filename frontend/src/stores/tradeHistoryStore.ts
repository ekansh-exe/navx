import { create } from "zustand";
import type { Card, Transaction } from "@/types/api";

export interface TradeHistoryEntry {
  transaction: Transaction;
  feeTransaction: Transaction;
  card: Card;
}

interface TradeHistoryState {
  entries: TradeHistoryEntry[];
  addEntry: (entry: TradeHistoryEntry) => void;
}

const MAX_ENTRIES = 100;

// There's no GET-transactions endpoint anywhere in the API (confirmed in
// API_ENDPOINTS.md's quests section) — "recent trades" is therefore only
// ever this session's own confirmed executions, not a real history. Portfolio
// labels this "this session" rather than implying it's complete.
export const useTradeHistoryStore = create<TradeHistoryState>((set) => ({
  entries: [],
  addEntry: (entry) =>
    set((state) => ({ entries: [entry, ...state.entries].slice(0, MAX_ENTRIES) })),
}));
