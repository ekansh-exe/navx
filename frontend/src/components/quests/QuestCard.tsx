import { CheckCircle2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { useCountdown } from "@/hooks/useCountdown";
import { formatCurrency } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Quest } from "@/types/api";

// DESIGN_SPEC_REFINED.md section 6 ("Quests"): animated progress bar with
// "60%" and "3/5", reward, gold-pulse+checkmark on completion (no confetti),
// "Resets in HH:MM:SS" top-right.
export function QuestCard({ quest, justCompleted }: { quest: Quest; justCompleted: boolean }) {
  const countdown = useCountdown(quest.reset_at);
  const percent = Math.min(100, (quest.progress / quest.target_value) * 100);

  return (
    <Card className={cn(quest.completed && "border-gold", justCompleted && "animate-quest-complete")}>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-2">
          <span className="text-sm font-medium text-text">{quest.title}</span>
          <span className="font-mono text-xs text-text-muted">Resets in {countdown}</span>
        </div>

        <Progress value={percent} />

        <div className="flex items-center justify-between text-xs">
          <span className="font-mono text-text-muted">
            {quest.progress} / {quest.target_value} ({Math.round(percent)}%)
          </span>
          <span className="font-mono text-gold">+{formatCurrency(quest.reward_currency)}</span>
        </div>

        {quest.completed && (
          <div className="flex items-center gap-1.5 text-sm text-gold">
            <CheckCircle2 className="size-4" />
            Completed
          </div>
        )}
      </CardContent>
    </Card>
  );
}
