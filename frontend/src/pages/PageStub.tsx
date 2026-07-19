import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Placeholder used while pages are built out one at a time (Dashboard first,
// per the build plan) — proves routing/layout without implementing page logic.
export function PageStub({ title, note }: { title: string; note?: ReactNode }) {
  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <Card className="max-w-md text-center">
        <CardHeader>
          <CardTitle className="text-2xl">{title}</CardTitle>
        </CardHeader>
        <CardContent className="text-text-muted">
          {note ?? "This page is scaffolded and ready — implementation is next."}
        </CardContent>
      </Card>
    </div>
  );
}
