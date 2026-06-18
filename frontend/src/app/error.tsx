"use client";

import { ErrorPanel } from "@/components/reader/states";

export default function RouteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex h-screen items-center justify-center bg-bg">
      <ErrorPanel error={error} onRetry={reset} />
    </div>
  );
}
