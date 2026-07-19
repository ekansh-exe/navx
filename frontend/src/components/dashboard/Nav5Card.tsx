import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PriceChangeIndicator } from "./PriceChangeIndicator";
import { Sparkline } from "./Sparkline";
import { formatCurrency } from "@/lib/format";
import { usePriceTick, useSessionChange, useTickHistory } from "@/ws/priceTickStore";
import type { Card as CardData } from "@/types/api";

// DESIGN_SPEC_REFINED.md section 6: NAV5 "should dominate the page" — double
// width, gold border, sparkline, sector composition, live %, placed at top.
export function Nav5Card({ card }: { card: CardData }) {
  const navigate = useNavigate();
  const tick = usePriceTick(card.id);
  const history = useTickHistory(card.id);
  const changePercent = useSessionChange(card.id);
  const price = tick?.price ?? card.current_price;

  return (
    <Card
      interactive
      onClick={() => navigate(`/cards/${card.id}`)}
      className="border-gold bg-gradient-to-br from-accent-nav5-bg-from to-accent-nav5-bg-to hover:border-gold"
    >
      <CardHeader>
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <CardTitle className="text-xl text-gold">NAV5</CardTitle>
            <Badge variant="gold">INDEX</Badge>
          </div>
          <PriceChangeIndicator percent={changePercent} className="text-sm" />
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-5 md:flex-row md:items-center">
        <div className="flex flex-col gap-1 md:w-64 md:shrink-0">
          <span className="font-mono text-4xl leading-10 font-semibold text-text">
            {formatCurrency(price)}
          </span>
          <span className="text-sm text-text-muted">{card.name}</span>
        </div>

        <div className="min-w-0 flex-1">
          <Sparkline data={history} height={80} />
        </div>

        <div className="flex flex-col gap-2 md:w-56 md:shrink-0 md:border-l md:border-border md:pl-5">
          <span className="text-xs font-medium tracking-wide text-text-muted uppercase">
            Sector composition
          </span>
          {/* No endpoint exposes sector weights for the index today — see
              API_ENDPOINTS.md; showing an honest placeholder rather than
              fabricated percentages. */}
          <span className="text-sm text-text-disabled">Unavailable</span>
        </div>
      </CardContent>
    </Card>
  );
}
