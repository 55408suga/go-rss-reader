// Presentation-only derivations. None of these come from the backend; they are
// computed on the client purely for display, so they live as pure functions
// that are easy to unit-test and reason about.

import type { Feed } from "./api/schemas";

/** Japanese relative time, falling back to an absolute date for older items. */
export function relativeTime(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const diffMs = now.getTime() - then.getTime();
  if (Number.isNaN(diffMs)) return "";

  const sec = Math.floor(diffMs / 1000);
  if (sec < 45) return "たった今";

  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}分前`;

  const hour = Math.floor(min / 60);
  if (hour < 24 && then.getDate() === now.getDate()) {
    return `${hour}時間前`;
  }

  const startOfToday = new Date(now);
  startOfToday.setHours(0, 0, 0, 0);
  const dayDiff = Math.floor(
    (startOfToday.getTime() - new Date(then).setHours(0, 0, 0, 0)) / 86_400_000,
  );
  if (dayDiff <= 0) return `${hour}時間前`;
  if (dayDiff === 1) return "昨日";
  if (dayDiff < 7) return `${dayDiff}日前`;

  return `${then.getMonth() + 1}月${then.getDate()}日`;
}

/** Bare host without a leading www., e.g. "engineering.mercari.com". */
export function hostFromUrl(url: string): string {
  try {
    return new URL(url).host.replace(/^www\./, "");
  } catch {
    return url;
  }
}

// Roughly 500 Japanese characters per minute; English words average out close
// enough that a single character-based estimate is fine for a reading-time pill.
const CHARS_PER_MINUTE = 500;

export function readingMinutes(article: {
  content: string;
  description: string;
}): number {
  const text = article.content.length > 0 ? article.content : article.description;
  return Math.max(1, Math.round(text.length / CHARS_PER_MINUTE));
}

/** First meaningful letter of a feed title, for the avatar tile. */
export function feedInitial(title: string): string {
  const trimmed = title.replace(/^the\s+/i, "").trim();
  return trimmed.length > 0 ? trimmed[0].toUpperCase() : "?";
}

// Stable string hash (FNV-1a) so the same feed always maps to the same hue,
// even though the backend has no notion of a per-feed color.
function hashString(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** Deterministic 0..359 hue derived from a feed's identity. */
export function feedHue(seed: string): number {
  return hashString(seed) % 360;
}

/** oklch avatar color matching the handoff's light/dark lightness/chroma pair. */
export function avatarColor(hue: number, dark: boolean): string {
  return dark ? `oklch(0.66 0.15 ${hue})` : `oklch(0.6 0.14 ${hue})`;
}

/** Convenience: the avatar color for a feed, keyed off its id (stable). */
export function feedAvatarColor(feed: Pick<Feed, "id">, dark: boolean): string {
  return avatarColor(feedHue(feed.id), dark);
}
