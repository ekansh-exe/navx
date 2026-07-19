import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useQuests } from "@/hooks/useQuests";
import { QuestCard } from "@/components/quests/QuestCard";
import type { Quest } from "@/types/api";

function partitionQuests(quests: Quest[]) {
  const now = Date.now();
  // The API only ever returns the current quest set, no history
  // (API_ENDPOINTS.md) — "expired" here means the reset deadline has
  // already passed locally and a refetch just hasn't landed yet, not a
  // real archived-quests list.
  const expired = quests.filter((q) => new Date(q.reset_at).getTime() <= now);
  const live = quests.filter((q) => new Date(q.reset_at).getTime() > now);
  return {
    active: live.filter((q) => !q.completed),
    completed: live.filter((q) => q.completed),
    expired,
  };
}

function QuestSection({
  title,
  quests,
  justCompletedIds,
  emptyLabel,
}: {
  title: string;
  quests: Quest[];
  justCompletedIds: Set<string>;
  emptyLabel: string;
}) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold text-text">{title}</h2>
      {quests.length === 0 ? (
        <Card>
          <CardContent className="py-6 text-center text-sm text-text-muted">{emptyLabel}</CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {quests.map((quest) => (
            <QuestCard key={quest.id} quest={quest} justCompleted={justCompletedIds.has(quest.id)} />
          ))}
        </div>
      )}
    </div>
  );
}

// DESIGN_SPEC_REFINED.md section 6 ("Quests"): three sections — Active,
// Completed, Expired.
export function QuestsPage() {
  const { data, isLoading, isError, justCompletedIds } = useQuests();

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full rounded-card" />
        ))}
      </div>
    );
  }

  if (isError || !data) {
    return (
      <Card>
        <CardContent className="py-16 text-center text-text-muted">
          Quests are unavailable right now.
        </CardContent>
      </Card>
    );
  }

  const { active, completed, expired } = partitionQuests(data.quests);

  return (
    <div className="flex flex-col gap-8">
      <h1 className="text-2xl font-semibold text-text">Quests</h1>
      <QuestSection
        title="Active"
        quests={active}
        justCompletedIds={justCompletedIds}
        emptyLabel="No active quests right now."
      />
      <QuestSection
        title="Completed"
        quests={completed}
        justCompletedIds={justCompletedIds}
        emptyLabel="Nothing completed yet today."
      />
      <QuestSection
        title="Expired"
        quests={expired}
        justCompletedIds={justCompletedIds}
        emptyLabel="Nothing has expired."
      />
    </div>
  );
}
