import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { listQuests } from "@/api/quests";
import { formatCurrency } from "@/lib/format";

const REFETCH_INTERVAL = 30_000;
const CELEBRATION_MS = 900;

// GET /api/quests is live today. Completion + payout happen atomically
// server-side (API_ENDPOINTS.md) — there's no "claim" step, so "just
// completed" is detected by diffing this poll against the previous one.
export function useQuests() {
  const query = useQuery({
    queryKey: ["quests"],
    queryFn: listQuests,
    refetchInterval: REFETCH_INTERVAL,
  });

  const previousCompleted = useRef<Map<string, boolean>>(new Map());
  const [justCompletedIds, setJustCompletedIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!query.data) return;
    const prev = previousCompleted.current;
    const justCompleted = new Set<string>();

    for (const quest of query.data.quests) {
      const wasCompleted = prev.get(quest.id);
      if (wasCompleted === false && quest.completed) {
        justCompleted.add(quest.id);
        toast(`Quest complete: ${quest.title}`, {
          description: `+${formatCurrency(quest.reward_currency)} credited`,
        });
      }
    }

    previousCompleted.current = new Map(query.data.quests.map((q) => [q.id, q.completed]));

    if (justCompleted.size > 0) {
      setJustCompletedIds(justCompleted);
      setTimeout(() => setJustCompletedIds(new Set()), CELEBRATION_MS);
    }
  }, [query.data]);

  return { ...query, justCompletedIds };
}
