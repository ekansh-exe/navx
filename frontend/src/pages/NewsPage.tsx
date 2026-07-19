import { useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useNewsPage } from "@/hooks/useNewsPage";
import { NewsCategoryIcon } from "@/components/news/NewsCategoryIcon";
import { categorySentiment, extractAffectedSectors } from "@/lib/newsParsing";

const SENTIMENT_VARIANT = { bullish: "success", bearish: "danger", neutral: "outline" } as const;
const LIMIT = 20;

// DESIGN_SPEC_REFINED.md section 6 ("News"): newest item gets a blue "NEW"
// badge (only true on the very first page), newspaper-style feed. GET
// /api/news is live and paginated (limit/offset) today.
export function NewsPage() {
  const [offset, setOffset] = useState(0);
  const { data, isLoading, isError } = useNewsPage(offset, LIMIT);

  const hasNext = (data?.news.length ?? 0) === LIMIT;
  const hasPrev = offset > 0;

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold text-text">News</h1>

      <div className="flex flex-col divide-y divide-border rounded-card border border-border bg-surface">
        {isLoading &&
          Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2 p-5">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))}

        {isError && (
          <div className="p-10 text-center text-text-muted">News is unavailable right now.</div>
        )}

        {!isLoading && !isError && data?.news.length === 0 && (
          <div className="p-10 text-center text-text-muted">No news to show.</div>
        )}

        {data?.news.map((item, i) => {
          const sentiment = categorySentiment(item.category);
          const sectors = extractAffectedSectors(item.headline);
          const isNewest = offset === 0 && i === 0;

          return (
            <div key={item.id} className="flex gap-4 p-5">
              <NewsCategoryIcon category={item.category} className="mt-0.5 size-5 shrink-0 text-info" />
              <div className="flex min-w-0 flex-col gap-2">
                <div className="flex items-start gap-2">
                  <span className="text-sm text-text">{item.headline}</span>
                  {isNewest && <Badge variant="info" className="shrink-0">NEW</Badge>}
                </div>
                <div className="flex flex-wrap items-center gap-1.5 text-xs text-text-muted">
                  <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
                    {item.category}
                  </Badge>
                  <Badge variant={SENTIMENT_VARIANT[sentiment]} className="px-1.5 py-0 text-[10px] uppercase">
                    {sentiment}
                  </Badge>
                  {sectors.map((sector) => (
                    <Badge key={sector} variant="outline" className="px-1.5 py-0 text-[10px]">
                      {sector}
                    </Badge>
                  ))}
                  <span>{new Date(item.created_at).toLocaleString()}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {!isError && (data?.news.length ?? 0) > 0 && (
        <div className="flex items-center justify-between">
          <Button
            variant="secondary"
            size="sm"
            disabled={!hasPrev}
            onClick={() => setOffset((o) => Math.max(0, o - LIMIT))}
          >
            <ChevronLeft className="size-4" /> Previous
          </Button>
          <span className="text-xs text-text-muted">Showing from {offset + 1}</span>
          <Button
            variant="secondary"
            size="sm"
            disabled={!hasNext}
            onClick={() => setOffset((o) => o + LIMIT)}
          >
            Next <ChevronRight className="size-4" />
          </Button>
        </div>
      )}

      {isError && (
        <Card>
          <CardContent className="py-6 text-center text-text-muted">
            Check back shortly for market news.
          </CardContent>
        </Card>
      )}
    </div>
  );
}
