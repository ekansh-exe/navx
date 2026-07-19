import { useParams } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useCard, useCardPriceHistory } from "@/hooks/useCard";
import { useHoldings } from "@/hooks/useHoldings";
import { PriceChart } from "@/components/card-detail/PriceChart";
import { PositionCard } from "@/components/card-detail/PositionCard";
import { OrderPanel } from "@/components/card-detail/OrderPanel";
import { RelatedNews } from "@/components/card-detail/RelatedNews";
import { formatCurrency } from "@/lib/format";
import { usePriceTick } from "@/ws/priceTickStore";

const STATUS_VARIANT = { ACTIVE: "success", DELISTED: "outline", FROZEN: "warning" } as const;

// DESIGN_SPEC_REFINED.md section 6 ("Card Detail"): 65/35 chart/order-panel
// split, 520px chart, position never mixed with market stats, related news
// timeline below the chart.
export function CardDetailPage() {
  const { cardId } = useParams();
  const { data: card, isLoading, isError } = useCard(cardId);
  const priceHistory = useCardPriceHistory(cardId);
  const holdingsQuery = useHoldings();
  const tick = usePriceTick(cardId ?? "");

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-[65fr_35fr]">
          <Skeleton className="h-[520px] w-full rounded-card" />
          <Skeleton className="h-[520px] w-full rounded-card" />
        </div>
      </div>
    );
  }

  if (isError || !card) {
    return (
      <Card>
        <CardContent className="py-16 text-center text-text-muted">
          This card isn't available right now — the card-detail endpoint isn't live yet.
        </CardContent>
      </Card>
    );
  }

  const price = tick?.price ?? card.current_price;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-semibold text-text">
          {card.name} <span className="text-text-muted">({card.symbol})</span>
        </h1>
        <Badge variant={STATUS_VARIANT[card.status]}>{card.status}</Badge>
        <span className="ml-auto font-mono text-2xl font-semibold text-text">
          {formatCurrency(price)}
        </span>
      </div>

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[65fr_35fr]">
        <div className="flex flex-col gap-5">
          <PriceChart
            cardId={card.id}
            restTicks={priceHistory.data?.ticks}
            isLoading={priceHistory.isLoading}
            isError={priceHistory.isError}
          />
          <RelatedNews card={card} />
        </div>

        <div className="flex flex-col gap-5">
          <PositionCard card={card} isLoading={holdingsQuery.isLoading} />
          <OrderPanel card={card} />
        </div>
      </div>
    </div>
  );
}
