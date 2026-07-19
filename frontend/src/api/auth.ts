import { apiFetch } from "./client";
import type { LoginRequest, LoginResponse, RegisterRequest, User } from "@/types/api";

export function register(req: RegisterRequest) {
  return apiFetch<User>("/api/auth/register", { method: "POST", body: req });
}

export function login(req: LoginRequest) {
  return apiFetch<LoginResponse>("/api/auth/login", { method: "POST", body: req });
}

export function getMe() {
  return apiFetch<User>("/api/users/me", { auth: true });
}
