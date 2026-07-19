/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Both optional — api/client.ts and ws/WebSocketProvider.tsx fall back to
  // the standard local-dev backend (localhost:8080) when unset.
  readonly VITE_API_BASE_URL?: string;
  readonly VITE_WS_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
