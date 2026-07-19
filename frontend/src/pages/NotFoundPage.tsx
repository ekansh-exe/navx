import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-bg text-center">
      <span className="font-mono text-6xl font-semibold text-text-muted">404</span>
      <p className="text-text-secondary">This page doesn't exist.</p>
      <Button asChild>
        <Link to="/">Back to Market</Link>
      </Button>
    </div>
  );
}
