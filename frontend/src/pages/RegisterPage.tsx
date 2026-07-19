import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useLoginMutation, useRegisterMutation } from "@/hooks/useAuth";
import { validatePassword, validateUsername } from "@/lib/validation";
import { ApiRequestError } from "@/api/client";

export function RegisterPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const navigate = useNavigate();
  const registerMutation = useRegisterMutation();
  const loginMutation = useLoginMutation();

  const isPending = registerMutation.isPending || loginMutation.isPending;

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const usernameError = validateUsername(username);
    if (usernameError) return setFormError(usernameError);

    const passwordError = validatePassword(password);
    if (passwordError) return setFormError(passwordError);

    if (password !== confirmPassword) return setFormError("Passwords don't match");

    registerMutation.mutate(
      { username, password },
      {
        // Register doesn't return a token (API_ENDPOINTS.md) — chain an
        // immediate login with the same credentials for a one-step signup.
        onSuccess: () => {
          loginMutation.mutate(
            { username, password },
            { onSuccess: () => navigate("/", { replace: true }) }
          );
        },
      }
    );
  };

  const apiError =
    registerMutation.error instanceof ApiRequestError
      ? registerMutation.error.message
      : loginMutation.error instanceof ApiRequestError
        ? loginMutation.error.message
        : null;

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <div className="mb-2 flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-button bg-primary text-sm font-bold text-white">
              NX
            </span>
            <span className="text-lg font-semibold text-text">NavXchange</span>
          </div>
          <CardTitle className="text-2xl">Create an account</CardTitle>
          <CardDescription>Start with 1,000.00 currency.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="confirm-password">Confirm password</Label>
              <Input
                id="confirm-password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </div>

            {(formError ?? apiError) && (
              <p className="text-sm text-danger">{formError ?? apiError}</p>
            )}

            <Button type="submit" loading={isPending} loadingText="Creating account...">
              Create account
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-text-muted">
            Already have an account?{" "}
            <Link to="/login" className="text-primary hover:underline">
              Log in
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
