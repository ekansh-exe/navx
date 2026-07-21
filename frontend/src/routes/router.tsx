import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import { ProtectedRoute } from "./ProtectedRoute";

// Route-level code splitting: recharts/dashboard-only weight (largest chunk
// in the bundle) shouldn't ship on the login screen.
export const router = createBrowserRouter([
  {
    path: "/login",
    lazy: () => import("@/pages/LoginPage").then((m) => ({ Component: m.LoginPage })),
  },
  {
    path: "/register",
    lazy: () => import("@/pages/RegisterPage").then((m) => ({ Component: m.RegisterPage })),
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [
          {
            path: "/",
            lazy: () => import("@/pages/DashboardPage").then((m) => ({ Component: m.DashboardPage })),
          },
          {
            path: "/cards/:cardId",
            lazy: () => import("@/pages/CardDetailPage").then((m) => ({ Component: m.CardDetailPage })),
          },
          {
            path: "/portfolio",
            lazy: () => import("@/pages/PortfolioPage").then((m) => ({ Component: m.PortfolioPage })),
          },
          {
            path: "/leaderboard",
            lazy: () => import("@/pages/LeaderboardPage").then((m) => ({ Component: m.LeaderboardPage })),
          },
          {
            path: "/news",
            lazy: () => import("@/pages/NewsPage").then((m) => ({ Component: m.NewsPage })),
          },
          {
            path: "/quests",
            lazy: () => import("@/pages/QuestsPage").then((m) => ({ Component: m.QuestsPage })),
          },
          {
            path: "/rules",
            lazy: () => import("@/pages/RulesPage").then((m) => ({ Component: m.RulesPage })),
          },
        ],
      },
    ],
  },
  {
    path: "*",
    lazy: () => import("@/pages/NotFoundPage").then((m) => ({ Component: m.NotFoundPage })),
  },
]);
