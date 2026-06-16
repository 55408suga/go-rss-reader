"use client";

import { RefreshCw } from "lucide-react";
import { ApiError } from "@/lib/api/client";

/** Skeleton rows that mirror the article-row layout while data loads. */
export function TimelineSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div aria-hidden="true" className="animate-pulse">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex gap-[14px] px-6 py-4 border-b border-line">
          <div className="size-9 shrink-0 rounded-[10px] bg-line" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-32 rounded bg-line" />
            <div className="h-4 w-3/4 rounded bg-line" />
            <div className="h-3 w-full rounded bg-line" />
            <div className="h-3 w-20 rounded-full bg-line" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** Centered empty-state for zero feeds / zero articles. */
export function EmptyState({
  title,
  message,
  action,
}: {
  title: string;
  message: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      <h2 className="text-lg font-bold text-ink">{title}</h2>
      <p className="max-w-sm text-sm leading-relaxed text-mut">{message}</p>
      {action}
    </div>
  );
}

function describeError(error: unknown): { title: string; message: string } {
  if (error instanceof ApiError) {
    switch (error.code) {
      case "not_found":
        return {
          title: "見つかりませんでした",
          message: "対象のフィードまたは記事が存在しません。",
        };
      case "external_unavailable":
        return {
          title: "取得できませんでした",
          message:
            "上流の RSS 取得に失敗しました。少し時間をおいて再試行してください。",
        };
      case "invalid_argument":
        return {
          title: "リクエストが不正です",
          message: error.message,
        };
      default:
        return {
          title: "サーバーエラー",
          message:
            "サーバー側で問題が発生しました。時間をおいて再試行してください。",
        };
    }
  }
  return {
    title: "問題が発生しました",
    message:
      "データを読み込めませんでした。ネットワーク接続を確認して再試行してください。",
  };
}

/** Error fallback with a retry affordance. */
export function ErrorPanel({
  error,
  onRetry,
}: {
  error: unknown;
  onRetry?: () => void;
}) {
  const { title, message } = describeError(error);
  const requestId = error instanceof ApiError ? error.requestId : undefined;

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      <h2 className="text-lg font-bold text-ink">{title}</h2>
      <p className="max-w-sm text-sm leading-relaxed text-mut">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-1 inline-flex items-center gap-2 rounded-[10px] bg-orange px-4 py-2 text-sm font-bold text-white shadow-[0_5px_14px_rgba(247,147,30,0.32)]"
        >
          <RefreshCw size={15} />
          再試行
        </button>
      )}
      {requestId && <p className="text-xs text-mut2">request id: {requestId}</p>}
    </div>
  );
}
