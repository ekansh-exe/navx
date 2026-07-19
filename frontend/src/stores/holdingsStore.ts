import { create } from "zustand";
import type { Holding } from "@/types/api";

interface HoldingsState {
  byCardId: Record<string, Holding>;
  setHoldings: (holdings: Holding[]) => void;
  upsertHolding: (holding: Holding) => void;
  clear: () => void;
}

// Populated from GET /api/users/me/holdings (see api/holdings.ts — proposed
// endpoint) on login/app load, and refreshed after each confirmed trade.
// Never written to optimistically — DESIGN_SPEC_REFINED.md section 7 forbids
// updating balances/positions ahead of server confirmation.
export const useHoldingsStore = create<HoldingsState>((set) => ({
  byCardId: {},
  setHoldings: (holdings) =>
    set({
      byCardId: Object.fromEntries(holdings.map((h) => [h.card_id, h])),
    }),
  upsertHolding: (holding) =>
    set((state) => ({
      byCardId: { ...state.byCardId, [holding.card_id]: holding },
    })),
  clear: () => set({ byCardId: {} }),
}));
