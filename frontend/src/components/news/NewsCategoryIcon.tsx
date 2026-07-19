import {
  CloudRain,
  Sun,
  Swords,
  Ban,
  Users,
  Sparkles,
  Rocket,
  AlertTriangle,
  Newspaper,
  type LucideIcon,
} from "lucide-react";
import type { NewsCategory } from "@/types/api";

const ICONS: Record<string, LucideIcon> = {
  FLOOD: CloudRain,
  DROUGHT: Sun,
  WAR: Swords,
  EMBARGO: Ban,
  STRIKE: Users,
  DISCOVERY: Sparkles,
  CARD_LAUNCH: Rocket,
  CIRCUIT_BREAKER: AlertTriangle,
};

export function NewsCategoryIcon({ category, className }: { category: NewsCategory; className?: string }) {
  const Icon = ICONS[category] ?? Newspaper;
  return <Icon className={className} />;
}
