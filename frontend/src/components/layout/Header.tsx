import { Link } from "react-router-dom";
import { Menu, Wallet } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ConnectionIndicator } from "./ConnectionIndicator";
import { useAuthStore } from "@/stores/authStore";
import { useUiStore } from "@/stores/uiStore";
import { formatCurrency } from "@/lib/format";

// 68px header per DESIGN_SPEC_REFINED.md section 5. Balance never animates
// on tick (section 7) — it's a static mono readout, refreshed only from
// confirmed server responses.
export function Header() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const setMobileNavOpen = useUiStore((s) => s.setMobileNavOpen);

  return (
    <header className="fixed inset-x-0 top-0 z-30 flex h-header items-center gap-4 border-b border-border bg-surface px-4 md:px-6">
      <button
        type="button"
        className="rounded-button p-2 text-text-muted hover:bg-surface-hover hover:text-text md:hidden"
        onClick={() => setMobileNavOpen(true)}
        aria-label="Open navigation"
      >
        <Menu className="size-5" />
      </button>

      <Link to="/" className="flex items-center gap-2">
        <span className="flex size-8 items-center justify-center rounded-button bg-primary text-sm font-bold text-white">
          NX
        </span>
        <span className="hidden text-lg font-semibold tracking-tight text-text sm:inline">
          NavXchange
        </span>
      </Link>

      <div className="ml-auto flex items-center gap-4 md:gap-6">
        <ConnectionIndicator />

        {user && (
          <div className="flex items-center gap-2 rounded-button border border-border px-3 py-1.5">
            <Wallet className="size-4 text-text-muted" />
            <span className="font-mono text-sm font-medium text-text">
              {formatCurrency(user.currency_balance)}
            </span>
          </div>
        )}

        {user && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="gap-2">
                {user.username}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem disabled>{user.username}</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={logout} variant="destructive">
                Log out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </header>
  );
}
