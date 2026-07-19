import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { queryClient } from "@/lib/queryClient";
import { useLeaderboardStore } from "@/stores/leaderboardStore";
import { publishTick } from "./priceTickStore";
import type { ConnectionStatus, WsMessage } from "@/types/ws";
import type { NewsListResponse } from "@/types/api";

// Defaults to the standard local-dev backend so the app runs with zero .env
// setup — override VITE_WS_BASE_URL for anything else.
const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || "ws://localhost:8080";

const BASE_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 15000;

type SocketStatus = ConnectionStatus;

/**
 * Manages one WS channel with reconnect-with-backoff. The backend's
 * internal/ws layer doesn't exist yet (API_ENDPOINTS.md section 7) — this is
 * built against the proposed envelope so it degrades to "disconnected"
 * gracefully today and starts working the moment the server ships it, with
 * no frontend changes needed.
 */
function useManagedSocket(path: string, onMessage: (msg: WsMessage) => void) {
  const [status, setStatus] = useState<SocketStatus>("connecting");
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let attempt = 0;
    let cancelled = false;

    const connect = () => {
      if (cancelled) return;
      setStatus("connecting");
      socket = new WebSocket(`${WS_BASE_URL}${path}`);

      socket.onopen = () => {
        attempt = 0;
        setStatus("live");
      };

      socket.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data) as WsMessage;
          onMessageRef.current(parsed);
        } catch {
          // Malformed frame — ignore rather than crash the socket handler.
        }
      };

      const scheduleReconnect = () => {
        if (cancelled) return;
        setStatus("disconnected");
        const backoff = Math.min(BASE_BACKOFF_MS * 2 ** attempt, MAX_BACKOFF_MS);
        attempt += 1;
        reconnectTimer = setTimeout(connect, backoff);
      };

      socket.onclose = scheduleReconnect;
      socket.onerror = () => socket?.close();
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [path]);

  return status;
}

interface WebSocketContextValue {
  connectionStatus: SocketStatus;
}

const WebSocketContext = createContext<WebSocketContextValue>({
  connectionStatus: "disconnected",
});

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const setLeaderboardEntries = useLeaderboardStore((s) => s.setEntries);

  const pricesStatus = useManagedSocket("/ws/prices", (msg) => {
    if (msg.type === "price_tick") publishTick(msg.data);
  });

  const newsStatus = useManagedSocket("/ws/news", (msg) => {
    if (msg.type !== "news_published") return;
    const item = msg.data;

    queryClient.setQueryData<NewsListResponse>(["news"], (current) => {
      if (!current) return current;
      if (current.news.some((n) => n.id === item.id)) return current;
      return { ...current, news: [item, ...current.news] };
    });

    // DESIGN_SPEC_REFINED.md section 7: news toast, top-right, headline + view.
    toast(item.headline, {
      position: "top-right",
      description: item.category,
    });
  });

  // Proposed extension beyond SPEC.md's WS surface (API_ENDPOINTS.md section
  // 7) — not a committed channel, so it's expected to stay disconnected
  // until/unless the backend adds it.
  const leaderboardStatus = useManagedSocket("/ws/leaderboard", (msg) => {
    if (msg.type !== "leaderboard_update") return;
    setLeaderboardEntries(msg.data.leaderboard);
    queryClient.setQueryData(["leaderboard"], { leaderboard: msg.data.leaderboard });
  });

  const statuses = [pricesStatus, newsStatus, leaderboardStatus];
  const connectionStatus: SocketStatus = statuses.every((s) => s === "live")
    ? "live"
    : statuses.every((s) => s === "disconnected")
      ? "disconnected"
      : "connecting";

  return (
    <WebSocketContext.Provider value={{ connectionStatus }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useConnectionStatus() {
  return useContext(WebSocketContext).connectionStatus;
}
