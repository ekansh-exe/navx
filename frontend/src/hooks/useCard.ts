import { useQuery } from "@tanstack/react-query";
import { getCard, getPriceHistory } from "@/api/cards";

// GET /api/cards/{id} is proposed/NOT YET IMPLEMENTED — see API_ENDPOINTS.md.
export function useCard(cardId: string | undefined) {
  return useQuery({
    queryKey: ["card", cardId],
    queryFn: () => getCard(cardId!),
    enabled: !!cardId,
  });
}

// GET /api/cards/{id}/price-history is proposed/NOT YET IMPLEMENTED.
export function useCardPriceHistory(cardId: string | undefined) {
  return useQuery({
    queryKey: ["priceHistory", cardId],
    queryFn: () => getPriceHistory(cardId!, { limit: 500 }),
    enabled: !!cardId,
  });
}
