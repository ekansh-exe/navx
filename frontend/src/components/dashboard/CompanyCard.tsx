import { useNavigate } from "react-router-dom";
import { Card, CardContent } from "@/components/ui/card";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { PriceChangeIndicator } from "./PriceChangeIndicator";
import { Sparkline } from "./Sparkline";
import { formatCurrency, formatShares } from "@/lib/format";
import { usePriceTick, useSessionChange, useSessionVolume, useTickHistory } from "@/ws/priceTickStore";
import type { Card as CardData } from "@/types/api";

// DESIGN_SPEC_REFINED.md section 6: logo, symbol, price, daily %, mini
// sparkline, volume — no descriptions. `interactive` Card handles the hover
// lift/border per section 4.
export function CompanyCard({ card }: { card: CardData }) {
  const navigate = useNavigate();
  const tick = usePriceTick(card.id);
  const history = useTickHistory(card.id);
  const changePercent = useSessionChange(card.id);
  const sessionVolume = useSessionVolume(card.id);
  const price = tick?.price ?? card.current_price;

  return (
    <Card interactive onClick={() => navigate(`/cards/${card.id}`)}>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <Avatar className="size-8 rounded-button">
            {card.image_url && <AvatarImage src={card.image_url} alt="" />}
            <AvatarFallback className="rounded-button bg-surface-elevated text-xs font-semibold text-text-secondary">
              {card.symbol.slice(0, 2)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold text-text">{card.symbol}</div>
            <div className="truncate text-xs text-text-muted">{card.name}</div>
          </div>
        </div>

        <div className="flex items-baseline justify-between">
          <span className="font-mono text-lg font-medium text-text">{formatCurrency(price)}</span>
          <PriceChangeIndicator percent={changePercent} />
        </div>

        <Sparkline data={history} height={32} />

        <div className="flex items-center justify-between text-xs text-text-muted">
          <span>Volume</span>
          <span className="font-mono">{formatShares(sessionVolume)}</span>
        </div>
      </CardContent>
    </Card>
  );
}
