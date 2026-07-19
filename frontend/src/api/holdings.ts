import { apiFetch } from "./client";
import type { HoldingsResponse } from "@/types/api";

// NOT YET IMPLEMENTED anywhere (not even flagged in API_ENDPOINTS.md) — see
// types/api.ts Holding comment. Proposed endpoint, built ahead of the backend.
export function getMyHoldings() {
  return apiFetch<HoldingsResponse>("/api/users/me/holdings", { auth: true });
}
