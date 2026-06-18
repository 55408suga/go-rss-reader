// Route-level fallback shown during navigation/suspense. Static markup only
// (server component) — a sidebar + timeline skeleton shell that matches the app
// layout so the transition does not jump.
export default function Loading() {
  return (
    <div className="flex h-screen overflow-hidden">
      <aside className="flex w-[252px] shrink-0 flex-col gap-[15px] border-r border-line bg-tint p-[18px_14px]">
        <div className="flex items-center gap-[11px] px-1">
          <div className="size-[38px] shrink-0 rounded-[10px] bg-line" />
          <div className="h-5 w-24 rounded bg-line" />
        </div>
        <div className="h-10 rounded-[10px] bg-line" />
        <div className="flex animate-pulse flex-col gap-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-7 rounded-[9px] bg-line" />
          ))}
        </div>
      </aside>
      <main className="flex min-w-0 flex-1 flex-col px-6 pt-[18px]">
        <div className="h-6 w-40 rounded bg-line" />
        <div className="mt-[14px] h-[3px] rounded-[3px] bg-line" />
        <div className="mt-4 flex animate-pulse flex-col gap-6">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex gap-[14px]">
              <div className="size-9 shrink-0 rounded-[10px] bg-line" />
              <div className="flex-1 space-y-2">
                <div className="h-3 w-32 rounded bg-line" />
                <div className="h-4 w-3/4 rounded bg-line" />
                <div className="h-3 w-full rounded bg-line" />
              </div>
            </div>
          ))}
        </div>
      </main>
    </div>
  );
}
