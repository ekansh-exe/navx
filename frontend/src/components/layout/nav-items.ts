import type { LucideIcon } from "lucide-react";
import { LayoutGrid, Wallet, Trophy, Newspaper, ListChecks, ScrollText } from "lucide-react";

export interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
}

export const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "Market", icon: LayoutGrid },
  { to: "/portfolio", label: "Portfolio", icon: Wallet },
  { to: "/leaderboard", label: "Leaderboard", icon: Trophy },
  { to: "/news", label: "News", icon: Newspaper },
  { to: "/quests", label: "Quests", icon: ListChecks },
  { to: "/rules", label: "Rules", icon: ScrollText },
];
