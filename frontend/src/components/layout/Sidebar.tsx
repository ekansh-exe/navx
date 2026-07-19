import { NavLink } from "react-router-dom";
import { ChevronsLeft, ChevronsRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { useUiStore } from "@/stores/uiStore";
import { NAV_ITEMS } from "./nav-items";

function NavLinks({ collapsed, onNavigate }: { collapsed: boolean; onNavigate?: () => void }) {
  return (
    <nav className="flex flex-col gap-1 p-3">
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          end={to === "/"}
          onClick={onNavigate}
          className={({ isActive }) =>
            cn(
              "flex items-center gap-3 rounded-button px-3 py-2.5 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text",
              isActive && "bg-surface-hover text-primary",
              collapsed && "justify-center"
            )
          }
        >
          <Icon className="size-5 shrink-0" />
          {!collapsed && <span>{label}</span>}
        </NavLink>
      ))}
    </nav>
  );
}

// Desktop: fixed 296px sidebar. Tablet: collapsible to icon rail. Mobile:
// nav moves to a bottom drawer instead (see MobileNavDrawer).
// DESIGN_SPEC_REFINED.md section 6 ("Sidebar").
export function Sidebar() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);

  return (
    <aside
      className={cn(
        "fixed inset-y-0 top-header hidden flex-col border-r border-border bg-surface transition-[width] duration-[220ms] md:flex",
        collapsed ? "w-16" : "w-sidebar"
      )}
    >
      <div className="flex-1 overflow-y-auto">
        <NavLinks collapsed={collapsed} />
      </div>
      <button
        type="button"
        onClick={toggleSidebar}
        className="flex items-center justify-center gap-2 border-t border-border p-3 text-text-muted hover:bg-surface-hover hover:text-text lg:hidden"
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      >
        {collapsed ? <ChevronsRight className="size-4" /> : <ChevronsLeft className="size-4" />}
      </button>
    </aside>
  );
}

export function MobileNavLinks({ onNavigate }: { onNavigate: () => void }) {
  return <NavLinks collapsed={false} onNavigate={onNavigate} />;
}
