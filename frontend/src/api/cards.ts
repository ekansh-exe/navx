import { apiFetch } from "./client";
import type {
  Card,
  CardListResponse,
  LaunchCardRequest,
  LaunchCardResponse,
  PriceHistoryResponse,
} from "@/types/api";

// GET /api/cards is marked NOT YET IMPLEMENTED in API_ENDPOINTS.md — built
// against the documented proposed shape so this is ready the moment the
// backend ships it.
export function listCards(params?: { limit?: number; offset?: number }) {
  return apiFetch<CardListResponse>("/api/cards", { query: params });
}

// NOT YET IMPLEMENTED — proposed shape, single cardDTO.
export function getCard(id: string) {
  return apiFetch<Card>(`/api/cards/${id}`);
}

// NOT YET IMPLEMENTED — proposed shape, oldest→newest ticks.
export function getPriceHistory(id: string, params?: { limit?: number; offset?: number }) {
  return apiFetch<PriceHistoryResponse>(`/api/cards/${id}/price-history`, { query: params });
}

export function launchCard(req: LaunchCardRequest) {
  return apiFetch<LaunchCardResponse>("/api/cards", { method: "POST", body: req, auth: true });
}
