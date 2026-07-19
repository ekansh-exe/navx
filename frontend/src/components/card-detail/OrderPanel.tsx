import { useEffect, useState } from "react";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useTradeQuote } from "@/hooks/useTradeQuote";
import { useExecuteTrade } from "@/hooks/useExecuteTrade";
import { ApiRequestError } from "@/api/client";
import { formatCurrency } from "@/lib/format";
import type { Card as CardData, TradeType } from "@/types/api";

// DESIGN_SPEC_REFINED.md section 4 ("Order Panel"): highlighted quote
// preview (bg #162033, green left border), current price / estimated cost /
// fees / slippage / 8s quote expiry. Section 6 ("Executing Trade"): swap the
// button for "Executing..." + progress bar + estimated completion, disable
// all inputs — never an optimistic balance update (section 7).
export function OrderPanel({ card }: { card: CardData }) {
  const [tradeType, setTradeType] = useState<TradeType>("BUY");
  const [sharesInput, setSharesInput] = useState("");
  const [executingProgress, setExecutingProgress] = useState(0);
  const shares = Number(sharesInput) || 0;

  const quoteState = useTradeQuote(card.id, tradeType, shares);
  const executeTrade = useExecuteTrade();

  useEffect(() => {
    if (!executeTrade.isPending) {
      setExecutingProgress(0);
      return;
    }
    setExecutingProgress(15);
    const interval = setInterval(() => {
      setExecutingProgress((p) => (p < 90 ? p + (90 - p) * 0.25 : p));
    }, 200);
    return () => clearInterval(interval);
  }, [executeTrade.isPending]);

  const canBuy = card.status === "ACTIVE";
  const canSubmit = shares > 0 && !!quoteState.quote && !executeTrade.isPending;

  const handleSubmit = () => {
    executeTrade.mutate(
      { card, type: tradeType, shares },
      {
        onSuccess: () => setSharesInput(""),
      }
    );
  };

  const executeError =
    executeTrade.error instanceof ApiRequestError ? executeTrade.error.message : null;

  return (
    <div className="flex flex-col gap-4 rounded-card border border-border bg-surface p-5">
      <Tabs value={tradeType} onValueChange={(v) => setTradeType(v as TradeType)}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="BUY" disabled={!canBuy}>
            BUY
          </TabsTrigger>
          <TabsTrigger value="SELL">SELL</TabsTrigger>
        </TabsList>
      </Tabs>

      {!canBuy && tradeType === "BUY" && (
        <p className="text-xs text-warning">This card isn't active — buying is disabled.</p>
      )}

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="shares">Shares</Label>
        <Input
          id="shares"
          type="number"
          min={0}
          step={1}
          inputMode="numeric"
          value={sharesInput}
          disabled={executeTrade.isPending}
          onChange={(e) => setSharesInput(e.target.value)}
          placeholder="0"
        />
      </div>

      {shares > 0 && (
        <div className="flex flex-col gap-2 rounded-button border-l-2 border-success bg-[#162033] p-4 text-sm">
          <div className="flex justify-between text-text-muted">
            <span>Current price</span>
            <span className="font-mono text-text">{formatCurrency(card.current_price)}</span>
          </div>

          {quoteState.isLoading && <p className="text-text-muted">Fetching quote…</p>}
          {quoteState.error && <p className="text-danger">{quoteState.error}</p>}

          {quoteState.quote && (
            <>
              <div className="flex justify-between text-text-muted">
                <span>Avg. price / share</span>
                <span className="font-mono text-text">
                  {formatCurrency(quoteState.quote.estimated_price_per_share)}
                </span>
              </div>
              <div className="flex justify-between text-text-muted">
                <span>Fee</span>
                <span className="font-mono text-text">
                  {formatCurrency(quoteState.quote.estimated_fee)}
                </span>
              </div>
              <div className="flex justify-between font-medium">
                <span className="text-text">
                  {tradeType === "BUY" ? "Estimated cost" : "You'll receive"}
                </span>
                <span className="font-mono text-text">
                  {formatCurrency(Math.abs(quoteState.quote.estimated_cost))}
                </span>
              </div>
              <div className="text-xs text-text-disabled">
                Quote expires in {quoteState.secondsRemaining}s
              </div>
            </>
          )}
        </div>
      )}

      {executeTrade.isPending ? (
        <div className="flex flex-col gap-2">
          <Button variant={tradeType === "BUY" ? "buy" : "sell"} size="lg" loading loadingText="Executing..." />
          <Progress value={executingProgress} />
          <p className="text-center text-xs text-text-muted">Estimated completion: a few seconds</p>
        </div>
      ) : (
        <Button
          variant={tradeType === "BUY" ? "buy" : "sell"}
          size="lg"
          disabled={!canSubmit || (tradeType === "BUY" && !canBuy)}
          onClick={handleSubmit}
        >
          {tradeType} {card.symbol}
        </Button>
      )}

      {executeError && <p className="text-sm text-danger">{executeError}</p>}
    </div>
  );
}
