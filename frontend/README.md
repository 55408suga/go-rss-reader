# frontend — go-rss-reader リーダー UI

`go-rss-reader` のフロントエンド(Signal リーダー UI)です。バックエンド API(`/api/v1`)を
TanStack Query 経由で呼び出し、フィードと記事を閲覧します。プロジェクト全体の概要・API 仕様・
構成は、リポジトリルートの [`../README.md`](../README.md) を参照してください。

## 技術スタック

- Next.js 15.5(App Router / Turbopack)
- React 19
- TanStack Query(データ取得・キャッシュ)
- Zod(API レスポンスのスキーマ検証)
- Tailwind CSS v4
- lucide-react(アイコン)
- Vitest(ユニット)/ Playwright(E2E)

> 既読・スター・新着の状態はフロントエンド側で保持します(バックエンドには永続化されません)。

## 起動

```bash
pnpm install
pnpm dev          # http://localhost:3000
```

接続先 API は環境変数 `NEXT_PUBLIC_API_BASE_URL` で指定します(未設定時は `http://localhost:8080`)。
バックエンドの起動方法は [`../README.md`](../README.md) のクイックスタートを参照してください。

## テスト / 各種コマンド

```bash
pnpm test                 # Vitest(ユニット)
pnpm test:e2e:install     # 初回のみ: Playwright 用 Chromium を取得
pnpm test:e2e             # Playwright(E2E)
pnpm lint                 # ESLint
pnpm typecheck            # tsc --noEmit
pnpm build                # 本番ビルド
```

## ディレクトリ構成

```
src/
├── app/                  # App Router(layout / page / providers / error / loading / globals.css)
├── components/reader/    # リーダー UI(sidebar / timeline / article-row / add-feed-dialog ...)
└── lib/
    ├── api/              # API クライアント(client.ts)とスキーマ(schemas.ts)
    ├── hooks.ts          # TanStack Query フック
    ├── reader-store.tsx  # 既読/スター/新着のクライアント状態
    ├── theme.tsx
    └── format.ts
```
