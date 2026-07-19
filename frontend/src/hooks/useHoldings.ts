import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { getMyHoldings } from "@/api/holdings";
import { useHoldingsStore } from "@/stores/holdingsStore";

// GET /api/users/me/holdings is proposed, not implemented anywhere yet (see
// api/holdings.ts) — this will error/degrade gracefully until it exists.
export function useHoldings() {
  const setHoldings = useHoldingsStore((s) => s.setHoldings);
  const query = useQuery({
    queryKey: ["holdings"],
    queryFn: getMyHoldings,
  });

  useEffect(() => {
    if (query.data) setHoldings(query.data.holdings);
  }, [query.data, setHoldings]);

  return query;
}
