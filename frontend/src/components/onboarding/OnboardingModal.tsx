import { LayoutGrid, ArrowLeftRight, Wallet, Trophy, ScrollText } from "lucide-react";
import { Link } from "react-router-dom";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useOnboardingStore } from "@/stores/onboardingStore";

const STEPS = [
  {
    icon: LayoutGrid,
    title: "The Market",
    body: "Browse ~30 companies and the NAV5 index. Prices move live, driven by real trades, not a script.",
  },
  {
    icon: ArrowLeftRight,
    title: "Buying & selling",
    body: "Open a card and use the order panel. Every quote shows the estimated cost including fees, and expires after 8 seconds since the price can move before you confirm.",
  },
  {
    icon: Wallet,
    title: "Your portfolio",
    body: "Track your positions, cash, and net worth on the Portfolio tab as you trade.",
  },
  {
    icon: Trophy,
    title: "Leaderboard & quests",
    body: "Climb the leaderboard by net worth, and pick up daily quests for extra rewards.",
  },
];

// Shown once, right after a fresh registration (see RegisterPage +
// onboardingStore) — a quick tour of the core loop, not an exhaustive rules
// dump. Anyone wanting the actual trading rules (fees, position limits,
// circuit breakers) is pointed at the permanent Rules tab instead of having
// them crammed in here.
export function OnboardingModal() {
  const showOnboarding = useOnboardingStore((s) => s.showOnboarding);
  const dismissOnboarding = useOnboardingStore((s) => s.dismissOnboarding);

  return (
    <Dialog open={showOnboarding} onOpenChange={(open) => !open && dismissOnboarding()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Welcome to NavXchange</DialogTitle>
          <DialogDescription>A quick tour before you start trading.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {STEPS.map(({ icon: Icon, title, body }) => (
            <div key={title} className="flex gap-3">
              <span className="flex size-9 shrink-0 items-center justify-center rounded-button bg-surface-hover text-primary">
                <Icon className="size-5" />
              </span>
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-medium text-text">{title}</span>
                <span className="text-sm text-text-muted">{body}</span>
              </div>
            </div>
          ))}

          <div className="flex gap-3 rounded-button border border-border bg-surface-hover p-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-button bg-surface text-primary">
              <ScrollText className="size-5" />
            </span>
            <div className="flex flex-col gap-0.5">
              <span className="text-sm font-medium text-text">Trading rules apply</span>
              <span className="text-sm text-text-muted">
                Fees, position limits, and circuit breakers apply to every trade. See the{" "}
                <Link to="/rules" className="text-primary hover:underline" onClick={dismissOnboarding}>
                  Rules
                </Link>{" "}
                tab anytime.
              </span>
            </div>
          </div>
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button onClick={dismissOnboarding}>Got it, let's trade</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
