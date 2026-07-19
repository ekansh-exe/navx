import { useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useLoginMutation } from "@/hooks/useAuth";
import { ApiRequestError } from "@/api/client";

export function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const navigate = useNavigate();
  const location = useLocation();
  const loginMutation = useLoginMutation();

  const from = (location.state as { from?: Location })?.from?.pathname ?? "/";

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    loginMutation.mutate(
      { username, password },
      {
        onSuccess: (data) => {
          if (data.reward_granted) {
            toast(`+${data.reward_amount / 100} daily login reward!`, {
              description: `${data.user.login_streak_count} day streak`,
            });
          }
          navigate(from, { replace: true });
        },
      }
    );
  };

  const errorMessage =
    loginMutation.error instanceof ApiRequestError ? loginMutation.error.message : null;

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
          <CardTitle className="text-2xl">Log in</CardTitle>
          <CardDescription>Trade the market. Climb the leaderboard.</CardDescription>
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
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            {errorMessage && <p className="text-sm text-danger">{errorMessage}</p>}

            <Button type="submit" loading={loginMutation.isPending} loadingText="Logging in...">
              Log in
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-text-muted">
            No account?{" "}
            <Link to="/register" className="text-primary hover:underline">
              Create one
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
