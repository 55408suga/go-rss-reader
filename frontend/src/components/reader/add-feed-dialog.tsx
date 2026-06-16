"use client";

import { Check, Loader2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api/client";
import type { FeedCandidate } from "@/lib/api/schemas";
import { useDiscoverFeed, useRegisterFeed } from "@/lib/hooks";
import { hostFromUrl } from "@/lib/format";

type Mode = "url" | "discover";

export function AddFeedDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const [mode, setMode] = useState<Mode>("url");
  const [value, setValue] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [generalError, setGeneralError] = useState<string | null>(null);
  const [candidates, setCandidates] = useState<FeedCandidate[] | null>(null);
  const [subscribedUrl, setSubscribedUrl] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const register = useRegisterFeed();
  const discover = useDiscoverFeed();
  const pending = register.isPending || discover.isPending;

  // Reset and focus whenever the dialog opens.
  useEffect(() => {
    if (!open) return;
    setMode("url");
    setValue("");
    setFieldError(null);
    setGeneralError(null);
    setCandidates(null);
    setSubscribedUrl(null);
    const id = setTimeout(() => inputRef.current?.focus(), 0);
    return () => clearTimeout(id);
  }, [open]);

  // Close on Escape.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  function handleError(err: unknown, field: string) {
    if (err instanceof ApiError) {
      const fields = err.fieldErrors();
      if (fields[field]) {
        setFieldError(fields[field]);
        return;
      }
      switch (err.code) {
        case "conflict":
          setGeneralError("このフィード／サイトは既に登録されています。");
          return;
        case "not_found":
          setGeneralError("フィードが見つかりませんでした。");
          return;
        case "external_unavailable":
          setGeneralError(
            "取得できませんでした。URL を確認して再試行してください。",
          );
          return;
        default:
          setGeneralError(err.message);
          return;
      }
    }
    setGeneralError("予期しないエラーが発生しました。");
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    setFieldError(null);
    setGeneralError(null);
    const url = value.trim();
    if (!url) {
      setFieldError("URL を入力してください。");
      return;
    }

    if (mode === "url") {
      register.mutate(url, {
        onSuccess: () => onClose(),
        onError: (err) => handleError(err, "feed_url"),
      });
    } else {
      discover.mutate(url, {
        onSuccess: (data) => {
          setCandidates(data.candidates);
          setSubscribedUrl(data.feed.feed_url);
        },
        onError: (err) => handleError(err, "website_url"),
      });
    }
  }

  function subscribeCandidate(feedUrl: string) {
    setGeneralError(null);
    register.mutate(feedUrl, {
      onSuccess: () => onClose(),
      onError: (err) => handleError(err, "feed_url"),
    });
  }

  const field = mode === "url" ? "feed_url" : "website_url";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="フィードを追加"
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-md rounded-2xl border border-line bg-panel p-6 shadow-xl"
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold text-ink">フィードを追加</h2>
          <button
            type="button"
            aria-label="閉じる"
            onClick={onClose}
            className="rounded-md p-1 text-mut hover:bg-tint"
          >
            <X size={18} />
          </button>
        </div>

        {/* Mode switch */}
        <div className="mb-4 flex gap-1 rounded-[10px] bg-tint p-1 text-[13px] font-semibold">
          <ModeTab
            active={mode === "url"}
            onClick={() => {
              setMode("url");
              setFieldError(null);
              setGeneralError(null);
            }}
          >
            フィード URL
          </ModeTab>
          <ModeTab
            active={mode === "discover"}
            onClick={() => {
              setMode("discover");
              setFieldError(null);
              setGeneralError(null);
            }}
          >
            サイト URL から検出
          </ModeTab>
        </div>

        <form onSubmit={submit}>
          <label
            htmlFor="add-feed-input"
            className="mb-1 block text-[13px] font-semibold text-mut"
          >
            {mode === "url" ? "フィード URL" : "ウェブサイト URL"}
          </label>
          <input
            id="add-feed-input"
            ref={inputRef}
            type="url"
            inputMode="url"
            name={field}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={
              mode === "url"
                ? "https://example.com/feed.xml"
                : "https://example.com/blog"
            }
            aria-invalid={fieldError ? true : undefined}
            className="w-full rounded-[10px] border border-line bg-bg px-3 py-2 text-sm text-ink outline-none focus:border-cyan"
          />
          {fieldError && (
            <p className="mt-1 text-[12px] text-orange">{fieldError}</p>
          )}

          <button
            type="submit"
            disabled={pending}
            className="mt-4 flex w-full items-center justify-center gap-2 rounded-[10px] bg-orange px-3 py-[11px] text-[13.5px] font-bold text-white shadow-[0_5px_14px_rgba(247,147,30,0.32)] disabled:opacity-70"
          >
            {pending && <Loader2 size={16} className="animate-spin" />}
            {mode === "url" ? "登録する" : "検出して購読"}
          </button>
        </form>

        {generalError && (
          <p className="mt-3 rounded-[10px] bg-orange-t px-3 py-2 text-[12.5px] text-orange">
            {generalError}
          </p>
        )}

        {/* Discovered candidates */}
        {candidates && candidates.length > 0 && (
          <div className="mt-5">
            <p className="mb-2 text-[12px] font-semibold text-mut">
              検出されたフィード
            </p>
            <ul className="flex flex-col gap-1">
              {candidates.map((c) => {
                const isSubscribed = c.feed_url === subscribedUrl;
                return (
                  <li
                    key={c.feed_url}
                    className="flex items-center gap-2 rounded-[9px] border border-line px-3 py-2"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[13px] font-semibold text-ink">
                        {c.title || hostFromUrl(c.feed_url)}
                      </p>
                      <p className="truncate text-[11.5px] text-mut">
                        {c.feed_url}
                      </p>
                    </div>
                    {isSubscribed ? (
                      <span className="flex items-center gap-1 text-[12px] font-semibold text-cyan-d">
                        <Check size={14} />
                        購読中
                      </span>
                    ) : (
                      <button
                        type="button"
                        disabled={pending}
                        onClick={() => subscribeCandidate(c.feed_url)}
                        className="rounded-[8px] border border-line px-2 py-1 text-[12px] font-semibold text-cyan-d disabled:opacity-70"
                      >
                        購読
                      </button>
                    )}
                  </li>
                );
              })}
            </ul>
            <button
              type="button"
              onClick={onClose}
              className="mt-3 w-full rounded-[10px] border border-line px-3 py-2 text-[13px] font-semibold text-mut"
            >
              完了
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function ModeTab({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex-1 rounded-[7px] px-3 py-[7px] ${
        active ? "bg-panel text-cyan-d shadow-sm" : "text-mut"
      }`}
    >
      {children}
    </button>
  );
}
