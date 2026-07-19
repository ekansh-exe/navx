import { useAuthStore } from "@/stores/authStore";
import type { ApiError } from "@/types/api";

// Defaults to the standard local-dev backend so the app runs with zero .env
// setup — override VITE_API_BASE_URL for anything else (staging, a
// non-default port, etc).
const BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

export class ApiRequestError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
  }
}

interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  auth?: boolean;
  query?: Record<string, string | number | undefined>;
}

// A 401 from any 🔒 route means "log in again" (token expired/invalid) —
// API_ENDPOINTS.md conventions section — not a bug to retry around.
function handleUnauthorized() {
  useAuthStore.getState().logout();
}

export async function apiFetch<T>(
  path: string,
  { method = "GET", body, auth = false, query }: RequestOptions = {}
): Promise<T> {
  const url = new URL(path, BASE_URL);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
  }

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (auth) {
    const token = useAuthStore.getState().token;
    if (token) headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    if (response.status === 401 && auth) handleUnauthorized();
    const message = (data as ApiError).error ?? "internal error";
    throw new ApiRequestError(response.status, message);
  }

  return data as T;
}

export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
