import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatCurrency } from "@/lib/format";

interface Rule {
  title: string;
  body: string;
}

const PRICING_RULES: Rule[] = [
  {
    title: "Prices move on a live curve",
    body: "Each card's price is driven by its circulating supply, not a fixed number. Buying pushes the price up, selling pushes it down, and a large order visibly slips worse than a small one at the same starting price.",
  },
  {
    title: "Quotes are estimates, not locked prices",
    body: "A quote previews the cost before you commit, but it expires after 8 seconds. The price can move in that window, so the executed cost may differ slightly from the quote.",
  },
  {
    title: "Fees",
    body: "A 1% fee applies to every buy and sell, with a small minimum fee on very small trades.",
  },
];

const SAFEGUARD_RULES: Rule[] = [
  {
    title: "Position limit: 25% of a card",
    body: "You can't buy your way past owning more than 25% of any single card's circulating supply. Selling is never blocked by this.",
  },
  {
    title: "Circuit breaker",
    body: "If a card's price moves more than 15% within one minute, trading on that card halts for 30 seconds before resuming.",
  },
  {
    title: "Wash-trade deterrent",
    body: "Reversing your own last trade on the same card within 5 minutes multiplies that trade's fee by 5x instead of blocking it outright.",
  },
];

const REWARD_RULES: Rule[] = [
  {
    title: "Starting balance",
    body: `Every new account starts with ${formatCurrency(100000)}.`,
  },
  {
    title: "Daily login reward",
    body: `Logging in each day grants a small reward, tracked as a login streak.`,
  },
  {
    title: "Daily quests",
    body: "Three quests reset every day: make 3 trades, hold any card for 24 hours, and reach rank 50 or better on the leaderboard. Each pays out a currency reward on completion.",
  },
  {
    title: "Leaderboard",
    body: "Ranked by net worth (cash plus the current value of everything you hold), refreshed on a schedule. Only human accounts are ranked.",
  },
];

function RuleSection({ heading, rules }: { heading: string; rules: Rule[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{heading}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {rules.map((rule) => (
          <div key={rule.title} className="flex flex-col gap-1">
            <span className="text-sm font-medium text-text">{rule.title}</span>
            <span className="text-sm text-text-muted">{rule.body}</span>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

export function RulesPage() {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold text-text">Rules</h1>
        <p className="text-sm text-text-muted">
          How pricing, safeguards, and rewards work on NavXchange.
        </p>
      </div>

      <RuleSection heading="Pricing & fees" rules={PRICING_RULES} />
      <RuleSection heading="Anti-exploit safeguards" rules={SAFEGUARD_RULES} />
      <RuleSection heading="Rewards & rankings" rules={REWARD_RULES} />
    </div>
  );
}
