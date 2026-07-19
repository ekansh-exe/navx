import type { NewsCategory } from "@/types/api";

// News events carry no sentiment or sector field (API_ENDPOINTS.md) — the
// only real signal available is the category itself and the headline text
// the generator already wrote ("... affects X and Y markets"). Both
// extractions below only ever surface data the API actually returned; they
// never invent a value for a specific headline.

const BEARISH_CATEGORIES = new Set(["FLOOD", "DROUGHT", "WAR", "EMBARGO", "STRIKE", "CIRCUIT_BREAKER"]);
const BULLISH_CATEGORIES = new Set(["DISCOVERY", "CARD_LAUNCH"]);

export type Sentiment = "bullish" | "bearish" | "neutral";

/** Category-level convention (e.g. "war" reads as bearish), not per-instance analysis. */
export function categorySentiment(category: NewsCategory): Sentiment {
  if (BEARISH_CATEGORIES.has(category)) return "bearish";
  if (BULLISH_CATEGORIES.has(category)) return "bullish";
  return "neutral";
}

/** Extracts the sector list from "... affects X, Y and Z markets" — a literal parse of the real headline, not a guess. */
export function extractAffectedSectors(headline: string): string[] {
  const match = headline.match(/affects (.+?) markets?/i);
  if (!match) return [];
  return match[1]
    .replace(/,/g, " and")
    .split(/\s+and\s+/i)
    .map((s) => s.trim())
    .filter(Boolean);
}
