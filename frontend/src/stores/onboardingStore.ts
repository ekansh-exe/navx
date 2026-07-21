import { create } from "zustand";

// Deliberately NOT persisted: this only tracks "did this browser tab just
// finish a fresh registration", not "has this account ever seen onboarding"
// (that would need a backend-tracked flag per user, out of scope for the
// simple version). RegisterPage sets this to true right before navigating to
// "/"; OnboardingModal (mounted once in AppShell) shows once and flips it
// back to false on dismiss.
interface OnboardingState {
  showOnboarding: boolean;
  triggerOnboarding: () => void;
  dismissOnboarding: () => void;
}

export const useOnboardingStore = create<OnboardingState>((set) => ({
  showOnboarding: false,
  triggerOnboarding: () => set({ showOnboarding: true }),
  dismissOnboarding: () => set({ showOnboarding: false }),
}));
