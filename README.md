# go-rss-reader

Web 上の RSS/Atom フィードを集約し、記事を時系列で読むための**個人向け RSS リーダー**です。購読しているサイトの新着を、広告やノイズに邪魔されず自分のペースでまとめて追えることを目指しています。**Go 製の API バックエンド**と **Next.js 製のリーダー UI** の 2 部構成で、両者は `/api/v1` の HTTP API のみで疎結合に連携します。あわせて、クリーンアーキテクチャや一貫した API 設計を実践する学習プロジェクトでもあります。

> ⚠️ **ステータス: 仮実装段階(WIP)** — フロントエンド・バックエンドの主要フローは動作しますが、一部は未完です(下記「ステータス」参照)。

---

## ステータス(仮実装段階)

主要なユースケース(フィード登録 → 記事取得 → 閲覧)は通しで動作します。一方、以下は**未完または暫定実装**です:

- **定期取得ジョブはスタブ** — `backend/internal/job` のスケジューラ枠組みはありますが、`DueFeedFetcher.Exec` の本体は未実装です。フィードの更新は現状 `POST /feeds/:id/refresh` の手動トリガで行います。
- **既読 / スター / 新着はフロントエンドのローカル状態** — バックエンドには永続化されません(ブラウザ側で判定・保持)。
- **認証・マルチユーザは未対応** — 単一利用を前提とした最小構成です。

---

## 主な機能

- **フィード登録**
  - フィード URL の直接登録(`POST /feeds`)
  - Web ページ URL からの**オートディスカバリ**(`<link rel="alternate">` を解析、`POST /feeds/discover`)
- **フィード管理** — 一覧(ページネーション)/ 取得 / 削除 / 手動再取得
- **記事閲覧** — 全フィード横断 / フィード別の一覧をキーセットページネーションで取得
- **Signal リーダー UI** — 既読・スター・新着をフロントエンドで管理する閲覧画面

---

## 技術スタック

### バックエンド (`backend/`)

| 分類 | 採用技術 |
| --- | --- |
| 言語 | Go 1.25 |
| Web フレームワーク | Echo v5 |
| DB / ドライバ | PostgreSQL 18 / pgx v5 |
| クエリ生成 | sqlc |
| RSS 解析 | gofeed |
| バリデーション | go-playground/validator |
| ID 採番 | google/uuid(UUIDv7) |
| ロギング | 標準 `log/slog` |
| テスト | 標準 `go test` + testcontainers-go(統合テスト) |
| 開発支援 | air(ホットリロード)/ golangci-lint |

### フロントエンド (`frontend/`)

| 分類 | 採用技術 |
| --- | --- |
| フレームワーク | Next.js 15.5(App Router / Turbopack) |
| UI ライブラリ | React 19 |
| データ取得 | TanStack Query |
| スキーマ検証 | Zod |
| スタイリング | Tailwind CSS v4 |
| アイコン | lucide-react |
| テスト | Vitest(ユニット)/ Playwright(E2E) |
| パッケージ管理 | pnpm(workspace) |

### インフラ / 開発フロー

Docker Compose / GitHub Actions(CI)/ lefthook(Git hooks)/ CodeRabbit(自動レビュー)

---

## クイックスタート

> 前提: Docker(+ Docker Compose)、Go 1.25、Node.js + pnpm。compose はバックエンドと DB のみを起動します。**フロントエンドは別途 `pnpm dev` で起動**してください。

### 1. バックエンド + DB(Docker・推奨)

リポジトリルートには開発用の `.env`(既定の開発用認証情報)が含まれています。そのまま起動できます:

```bash
docker compose up
```

- PostgreSQL 18 と、`air` によるホットリロード付きの API が起動します。
- API は `http://localhost:8080`(ルートは `/api/v1`)で待ち受けます。
- DB スキーマは初回起動時に `backend/internal/infra/persistence/postgres/sql/schema.sql` から自動適用されます。

### 2. フロントエンド

```bash
cd frontend
pnpm install
pnpm dev
```

- `http://localhost:3000` で開きます。
- 接続先 API は環境変数 `NEXT_PUBLIC_API_BASE_URL` で指定します(未設定時は `http://localhost:8080`)。

### Docker を使わずにバックエンドを動かす場合

```bash
cd backend
# DATABASE_URL を環境に設定してから
go run ./cmd
```

---

## 環境変数

### バックエンド

| 変数 | 必須 | 既定値 | 説明 |
| --- | --- | --- | --- |
| `DATABASE_URL` | ✅ | — | PostgreSQL 接続文字列 |
| `LOG_LEVEL` | | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | | `text` | `text` / `json` |
| `CORS_ALLOWED_ORIGINS` | | `http://localhost:3000` | 許可するブラウザ Origin(カンマ区切り) |

### Docker Compose 用(`.env`)

| 変数 | 説明 |
| --- | --- |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | Postgres コンテナの初期化に使用 |

### フロントエンド

| 変数 | 既定値 | 説明 |
| --- | --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | バックエンド API のベース URL |

---

## API 仕様

すべてのエンドポイントは `/api/v1` 配下です。

| メソッド | パス | 概要 | リクエスト | 成功時 |
| --- | --- | --- | --- | --- |
| `POST` | `/feeds` | フィード URL を登録 | `{ "feed_url": "..." }` | `201`(`feed`, `articles`) |
| `POST` | `/feeds/discover` | Web ページから feed をオートディスカバリして登録 | `{ "website_url": "..." }` | `201`(`feed`, `articles`, `candidates`) |
| `GET` | `/feeds` | フィード一覧 | `?cursor=&limit=`(既定 10 / 1–100) | `200`(`feeds`) |
| `GET` | `/feeds/:id` | フィード詳細 | — | `200`(`feed`) |
| `POST` | `/feeds/:id/refresh` | フィードを再取得 | — | `204` |
| `DELETE` | `/feeds/:id` | フィード削除 | — | `204` |
| `GET` | `/articles` | 記事一覧(全フィード横断) | `?cursor=&limit=` | `200`(`articles`) |
| `GET` | `/feeds/:feed_id/articles` | フィード別の記事一覧 | `?cursor=&limit=` | `200`(`articles`) |

> 登録系で受け付ける URL は `http` / `https` スキームのみです。

### 共通レスポンス形式

すべてのレスポンスは封筒(envelope)で統一されています。

**成功:**

```json
{
  "data": { "feeds": [ /* ... */ ] },
  "meta": {
    "request_id": "...",
    "pagination": { "next_cursor": "opaque-token", "has_more": true }
  }
}
```

- `meta.request_id` は `X-Request-ID` ヘッダと相関(ログ追跡用)。
- 一覧系のみ `meta.pagination` が付与されます。`next_cursor` はクライアントが中身を解釈しない不透明トークンで、`null` は「次ページなし」を意味します。

**エラー:**

```json
{
  "error": { "code": "invalid_argument", "message": "...", "details": [ { "field": "feed_url", "reason": "..." } ] },
  "meta": { "request_id": "..." }
}
```

| `error.code` | HTTP ステータス |
| --- | --- |
| `not_found` | 404 |
| `invalid_argument` | 400 |
| `conflict` | 409 |
| `external_unavailable` | 502 |
| `internal` | 500 |

> 5xx の `message` はクライアントには `"internal server error"` に置き換えられます(内部情報を漏らさないため)。

### リクエスト例

```bash
curl -X POST http://localhost:8080/api/v1/feeds \
  -H 'Content-Type: application/json' \
  -d '{"feed_url": "https://example.com/feed.xml"}'
```

---

## アーキテクチャ概要

クリーンアーキテクチャに倣い、依存は内側(ドメイン)に向きます。コンポジションルートは `backend/internal/di/container.go`。

- `domain/model` — エンティティ(`Feed` / `Article` / `FetchStatus` など、ID は UUIDv7)
- `domain/repository` — リポジトリ**インターフェース**(実装は infra 側)
- `usecase` — インタラクタ。HTTP も SQL も知らない
- `handler` — Echo ハンドラ。入力検証して usecase に委譲、`apperror` を返す
- `infra/` — `gateway`(gofeed + HTTP クライアント)/ `persistence`(sqlc + リポジトリ)/ `middleware`(エラーハンドラ等)/ `router` / `config` / `logger`
- `job` — 定期取得スケジューラ(本体はスタブ。上記「ステータス」参照)

> 詳細な設計は親ワークツリーの `../docs/internal/arch.md` / `../docs/internal/directory.md`、コーディング規約・トランザクション・エラー戦略は **[`CLAUDE.md`](./CLAUDE.md)** を参照してください(README は入口、詳細は各ドキュメントに委譲)。

---

## データモデル

| テーブル | 役割 | 主な制約 |
| --- | --- | --- |
| `feeds` | 登録フィード | `feed_url` / `website_url` は UNIQUE、PK は UUID |
| `articles` | 記事 | `UNIQUE(feed_id, external_id)`、`feed_id` は `feeds` への外部キー(ON DELETE CASCADE) |
| `feed_fetch_status` | フィードごとの取得状態(ETag / Last-Modified / 次回取得予定など) | PK は `feed_id` |

スキーマ定義: `backend/internal/infra/persistence/postgres/sql/schema.sql`

---

## ディレクトリ構成

```
go-rss-reader/
├── backend/                      # Go API(module: rss_reader)
│   ├── cmd/                      # エントリポイント(main)
│   └── internal/
│       ├── domain/{model,repository}
│       ├── usecase/
│       ├── handler/              # Echo ハンドラ
│       ├── infra/{gateway,persistence,middleware,router,config,logger}
│       ├── di/                   # コンポジションルート
│       ├── job/                  # 定期取得スケジューラ(WIP)
│       └── apperror/ applog/ apiresp/
├── frontend/                     # Next.js リーダー UI
│   └── src/
│       ├── app/                  # App Router(layout / page / providers ...)
│       ├── components/reader/    # リーダー UI コンポーネント
│       └── lib/{api}/            # API クライアント / フック / ストア
├── .github/                      # GitHub Actions(CI)
├── docker-compose.yaml           # backend + Postgres(frontend は含まない)
├── CLAUDE.md                     # 開発規約(正典)
└── LICENSE
```

---

## 開発コマンド

### バックエンド(`backend/` 配下で実行)

```bash
go test ./...                 # テスト(-race 推奨: go test -race ./...)
golangci-lint run             # Lint
golangci-lint fmt             # フォーマット
sqlc generate                 # sql/*.sql 変更後にコード再生成
```

### フロントエンド(`frontend/` 配下で実行)

```bash
pnpm test                     # Vitest(ユニット)
pnpm test:e2e                 # Playwright(E2E、初回は pnpm test:e2e:install)
pnpm lint                     # ESLint
pnpm typecheck                # tsc --noEmit
pnpm build                    # 本番ビルド
```

詳しいビルド/テスト方針・規約は [`CLAUDE.md`](./CLAUDE.md) を参照してください。

---

## ライセンス

[MIT License](./LICENSE) © 2026 55408suga
