import { useMutation } from "@tanstack/react-query";
import { login, register } from "@/api/auth";
import { useAuthStore } from "@/stores/authStore";
import type { LoginRequest, LoginResponse, RegisterRequest } from "@/types/api";

export function useLoginMutation() {
  const setSession = useAuthStore((s) => s.setSession);

  return useMutation({
    mutationFn: (req: LoginRequest) => login(req),
    onSuccess: (data: LoginResponse) => setSession(data.token, data.user),
  });
}

export function useRegisterMutation() {
  return useMutation({
    mutationFn: (req: RegisterRequest) => register(req),
  });
}
