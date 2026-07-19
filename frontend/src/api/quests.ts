import { apiFetch } from "./client";
import type { QuestListResponse } from "@/types/api";

export function listQuests() {
  return apiFetch<QuestListResponse>("/api/quests", { auth: true });
}
