import { create } from "zustand";

interface UiState {
  // Tablet breakpoint: sidebar collapses to icons-only, toggled by the user.
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  // Mobile breakpoint: nav lives in a bottom drawer instead of a sidebar.
  mobileNavOpen: boolean;
  setMobileNavOpen: (open: boolean) => void;
}

export const useUiStore = create<UiState>((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  mobileNavOpen: false,
  setMobileNavOpen: (open) => set({ mobileNavOpen: open }),
}));
