import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useNews } from "@/hooks/useNews";
import { categorySentiment, extractAffectedSectors } from "@/lib/newsParsing";
import type { Card as CardData } from "@/types/api";

const SENTIMENT_VARIANT = { bullish: "success", bearish: "danger", neutral: "outline" } as const;

// DESIGN_SPEC_REFINED.md section 6 ("Related News"): timeline below the
// chart — headline, bullish/bearish, sector, timestamp. `related_card_id` is
// almost always null today (API_ENDPOINTS.md), so this mostly falls back to
// general market news, clearly labeled as such rather than implied specific.
export function RelatedNews({ card }: { card: CardData }) {
  const { data, isLoading } = useNews(30);

  const directMatches = data?.news.filter((n) => n.related_card_id === card.id) ?? [];
  const items = directMatches.length > 0 ? directMatches : (data?.news ?? []).slice(0, 6);
  const isDirectMatch = directMatches.length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{isDirectMatch ? "Related News" : "Market News"}</CardTitle>
        {!isDirectMatch && !isLoading && items.length > 0 && (
          <p className="text-xs text-text-muted">
            No news is tagged to {card.symbol} specifically yet — showing recent market headlines.
          </p>
        )}
      </CardHeader>
      <CardContent className="flex flex-col divide-y divide-border">
        {isLoading &&
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="my-2 h-10 w-full" />)}

        {!isLoading && items.length === 0 && (
          <p className="py-3 text-sm text-text-muted">No news yet.</p>
        )}

        {items.map((item) => {
          const sentiment = categorySentiment(item.category);
          const sectors = extractAffectedSectors(item.headline);
          return (
            <div key={item.id} className="flex flex-col gap-1.5 py-3">
              <div className="flex items-start justify-between gap-2">
                <span className="text-sm text-text">{item.headline}</span>
                <Badge variant={SENTIMENT_VARIANT[sentiment]} className="shrink-0 uppercase">
                  {sentiment}
                </Badge>
              </div>
              <div className="flex flex-wrap items-center gap-1.5 text-xs text-text-muted">
                {sectors.map((sector) => (
                  <Badge key={sector} variant="outline" className="px-1.5 py-0 text-[10px]">
                    {sector}
                  </Badge>
                ))}
                <span>{new Date(item.created_at).toLocaleString()}</span>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
