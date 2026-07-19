import { Outlet } from "react-router-dom";
import { Header } from "./Header";
import { Sidebar, MobileNavLinks } from "./Sidebar";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useUiStore } from "@/stores/uiStore";
import { cn } from "@/lib/utils";

function MobileNavDrawer() {
  const open = useUiStore((s) => s.mobileNavOpen);
  const setOpen = useUiStore((s) => s.setMobileNavOpen);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetContent side="bottom" className="max-h-[80vh] border-border bg-surface p-0">
        <SheetHeader className="border-b border-border px-4 py-3">
          <SheetTitle className="text-text">Menu</SheetTitle>
        </SheetHeader>
        <MobileNavLinks onNavigate={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  );
}

// Content column offsets around the fixed header (68px) and sidebar (296px,
// collapses to 64px on tablet, hidden on mobile in favor of the bottom
// drawer) — DESIGN_SPEC_REFINED.md sections 5 & 6. Content itself caps at
// 1600px, centered.
export function AppShell() {
  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed);

  return (
    <div className="min-h-screen bg-bg">
      <Header />
      <Sidebar />
      <MobileNavDrawer />
      <main
        className={cn(
          "pt-header transition-[padding] duration-[220ms] md:pl-16",
          !sidebarCollapsed && "md:pl-sidebar"
        )}
      >
        <div className="mx-auto max-w-[1600px] p-4 md:p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
