import { useEffect } from "react";
import { useCards } from "@/hooks/useCards";
import { seedTick } from "@/ws/priceTickStore";
import { Nav5Card } from "@/components/dashboard/Nav5Card";
import { CompanyCard } from "@/components/dashboard/CompanyCard";
import { CardGridSkeleton } from "@/components/dashboard/CardGridSkeleton";
import { NewsPanel } from "@/components/dashboard/NewsPanel";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

// DESIGN_SPEC_REFINED.md section 6 ("Dashboard"): NAV5 hero at top, company
// card grid (4/3/2/1 columns desktop→mobile) with a news panel alongside.
export function DashboardPage() {
  const { data, isLoading, isError } = useCards();

  const cards = data?.cards ?? [];
  const nav5 = cards.find((c) => c.card_type === "INDEX");
  const companyCards = cards.filter((c) => c.card_type !== "INDEX");

  useEffect(() => {
    for (const card of data?.cards ?? []) {
      seedTick(card.id, card.current_price, card.created_at);
    }
  }, [data?.cards]);

  return (
    <div className="flex flex-col gap-8">
      {isLoading && (
        <Card className="h-40">
          <CardContent className="flex h-full items-center gap-6">
            <Skeleton className="h-10 w-40" />
            <Skeleton className="h-16 flex-1" />
          </CardContent>
        </Card>
      )}
      {nav5 && <Nav5Card card={nav5} />}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[1fr_340px]">
        <div>
          {isLoading && <CardGridSkeleton />}

          {isError && (
            <Card>
              <CardContent className="py-10 text-center text-text-muted">
                Market data is unavailable right now — check back shortly.
              </CardContent>
            </Card>
          )}

          {!isLoading && !isError && companyCards.length === 0 && (
            <Card>
              <CardContent className="py-10 text-center text-text-muted">
                No cards are trading yet.
              </CardContent>
            </Card>
          )}

          {companyCards.length > 0 && (
            <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {companyCards.map((card) => (
                <CompanyCard key={card.id} card={card} />
              ))}
            </div>
          )}
        </div>

        <NewsPanel />
      </div>
    </div>
  );
}
