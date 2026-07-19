import { useMutation, useQueryClient } from "@tanstack/react-query";
import { executeTrade } from "@/api/trades";
import { newIdempotencyKey } from "@/api/client";
import { useAuthStore } from "@/stores/authStore";
import { useTradeHistoryStore } from "@/stores/tradeHistoryStore";
import type { Card, ExecuteTradeResponse, TradeType } from "@/types/api";

interface ExecuteTradeInput {
  card: Card;
  type: TradeType;
  shares: number;
}

// DESIGN_SPEC_REFINED.md section 7: never update balance optimistically —
// this only writes to stores once the server has confirmed the trade.
export function useExecuteTrade() {
  const updateUser = useAuthStore((s) => s.updateUser);
  const addTradeHistory = useTradeHistoryStore((s) => s.addEntry);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ card, type, shares }: ExecuteTradeInput) =>
      executeTrade({
        card_id: card.id,
        type,
        shares,
        idempotency_key: newIdempotencyKey(),
      }),
    onSuccess: (data: ExecuteTradeResponse) => {
      updateUser(data.user);
      addTradeHistory({
        transaction: data.transaction,
        feeTransaction: data.fee_transaction,
        card: data.card,
      });
      queryClient.setQueryData(["card", data.card.id], data.card);
      queryClient.setQueryData(["cards"], (current: { cards: Card[] } | undefined) => {
        if (!current) return current;
        return {
          ...current,
          cards: current.cards.map((c) => (c.id === data.card.id ? data.card : c)),
        };
      });
      // No optimistic holdings math — refetch the real thing (proposed
      // endpoint) rather than guess at the new avg cost basis client-side.
      queryClient.invalidateQueries({ queryKey: ["holdings"] });
    },
  });
}
