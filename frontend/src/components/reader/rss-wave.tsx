// The RSS broadcast glyph (dot + two arcs) — the Signal brand motif. Uses
// currentColor so callers tint it via text-orange.
export function RssWave({
  size = 14,
  className,
}: {
  size?: number;
  className?: string;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
      className={className}
    >
      <circle cx="5" cy="19" r="2.4" fill="currentColor" />
      <path
        d="M4 11.5a8.5 8.5 0 0 1 8.5 8.5"
        stroke="currentColor"
        strokeWidth="2.6"
        strokeLinecap="round"
      />
      <path
        d="M4 5a15 15 0 0 1 15 15"
        stroke="currentColor"
        strokeWidth="2.6"
        strokeLinecap="round"
      />
    </svg>
  );
}
