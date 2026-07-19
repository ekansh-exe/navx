import { useEffect, useRef, useState } from "react";
import { quoteTrade } from "@/api/trades";
import { ApiRequestError } from "@/api/client";
import type { QuoteResponse, TradeType } from "@/types/api";

const DEBOUNCE_MS = 400;
const QUOTE_TTL_SECONDS = 8;

interface QuoteState {
  quote: QuoteResponse | null;
  isLoading: boolean;
  error: string | null;
  secondsRemaining: number;
}

// DESIGN_SPEC_REFINED.md section 4 ("Order Panel"): quote expires in 8
// seconds. This is a non-binding preview (API_ENDPOINTS.md) — safe to call
// on every keystroke, debounced, and auto-refreshed while the panel is open
// so the countdown never blocks the user, only informs them the number may
// have moved.
export function useTradeQuote(cardId: string | undefined, type: TradeType, shares: number) {
  const [state, setState] = useState<QuoteState>({
    quote: null,
    isLoading: false,
    error: null,
    secondsRemaining: 0,
  });
  const requestId = useRef(0);

  useEffect(() => {
    if (!cardId || shares <= 0) {
      setState({ quote: null, isLoading: false, error: null, secondsRemaining: 0 });
      return;
    }

    let cancelled = false;
    const currentRequest = ++requestId.current;
    let countdownTimer: ReturnType<typeof setInterval> | undefined;

    const fetchQuote = async () => {
      setState((s) => ({ ...s, isLoading: true, error: null }));
      try {
        const quote = await quoteTrade({ card_id: cardId, type, shares });
        if (cancelled || requestId.current !== currentRequest) return;
        setState({ quote, isLoading: false, error: null, secondsRemaining: QUOTE_TTL_SECONDS });

        countdownTimer = setInterval(() => {
          setState((s) => {
            if (s.secondsRemaining <= 1) {
              fetchQuote();
              return s;
            }
            return { ...s, secondsRemaining: s.secondsRemaining - 1 };
          });
        }, 1000);
      } catch (err) {
        if (cancelled || requestId.current !== currentRequest) return;
        const message = err instanceof ApiRequestError ? err.message : "Unable to fetch quote";
        setState({ quote: null, isLoading: false, error: message, secondsRemaining: 0 });
      }
    };

    const debounceTimer = setTimeout(fetchQuote, DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(debounceTimer);
      if (countdownTimer) clearInterval(countdownTimer);
    };
  }, [cardId, type, shares]);

  return state;
}
