import { apiFetch } from "./client";
import type { NewsListResponse } from "@/types/api";

export function listNews(params?: { limit?: number; offset?: number }) {
  return apiFetch<NewsListResponse>("/api/news", { query: params });
}
