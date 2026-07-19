import { useState } from "react";
import { Newspaper } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Skeleton } from "@/components/ui/skeleton";
import { useNews } from "@/hooks/useNews";
import { cn } from "@/lib/utils";

function timeAgo(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function NewsItem({ headline, category, createdAt, isNewest }: {
  headline: string;
  category: string;
  createdAt: string;
  isNewest: boolean;
}) {
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex w-full flex-col gap-1 py-3 text-left">
        <div className="flex items-start justify-between gap-2">
          <span className={cn("text-sm text-text", open ? "" : "line-clamp-2")}>{headline}</span>
          {isNewest && (
            <Badge variant="info" className="shrink-0">
              NEW
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2 text-xs text-text-muted">
          <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
            {category}
          </Badge>
          <span>{timeAgo(createdAt)}</span>
        </div>
      </CollapsibleTrigger>
      <CollapsibleContent className="pb-2 text-xs text-text-muted">
        Affects sector-linked cards — open the card to see the price reaction.
      </CollapsibleContent>
    </Collapsible>
  );
}

// DESIGN_SPEC_REFINED.md section 6: top-right section, newest item gets a
// blue "NEW" badge, entries are expandable.
export function NewsPanel() {
  const { data, isLoading, isError } = useNews(15);

  return (
    <Card className="h-fit">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Newspaper className="size-4 text-info" />
          News
        </CardTitle>
      </CardHeader>
      <CardContent className="divide-y divide-border">
        {isLoading &&
          Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2 py-3">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-3 w-20" />
            </div>
          ))}

        {isError && <p className="py-3 text-sm text-text-muted">News unavailable right now.</p>}

        {data?.news.length === 0 && (
          <p className="py-3 text-sm text-text-muted">No news yet.</p>
        )}

        {data?.news.map((item, i) => (
          <NewsItem
            key={item.id}
            headline={item.headline}
            category={item.category}
            createdAt={item.created_at}
            isNewest={i === 0}
          />
        ))}
      </CardContent>
    </Card>
  );
}
