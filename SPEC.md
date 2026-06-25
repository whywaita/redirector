# redirector — SPEC

> **一言**: 受け取った HTTP リクエストを指定ドメインにリダイレクトするだけの極薄アプリケーション

---

## 1. コンセプト

- **単一責務**: リクエストを受け、設定されたリダイレクト先に転送する。それ以上も以下もやらない。
- **ゼロ依存（準）**: Go 標準ライブラリ (`net/http`) + Prometheus クライアント (`prometheus/client_golang`) のみ。
- **コンテナファースト**: Docker イメージで配布、Kubernetes 上で動作。

---

## 2. 設定

設定は **環境変数** と **CLI フラグ** の両方で受け付ける。CLI フラグが優先。

| 項目 | 環境変数 | CLI フラグ | デフォルト | 説明 |
|------|----------|-----------|------------|------|
| リダイレクト先 | `REDIRECT_DESTINATION` | `--destination` / `-d` | **必須（未設定時は起動エラー）** | 転送先ベース URL（スキーム含む） |
| ステータスコード | `REDIRECT_STATUS` | `--status` / `-s` | `302` | `301`, `302`, `307`, `308` のいずれか |
| 待受ポート | `PORT` | `--port` / `-p` | `8080` | 待受ポート番号 |

### 設定値の検証
- `REDIRECT_DESTINATION`: URL パース可能であること。空文字列 or 不正な場合は起動時に fatal エラー。
- `REDIRECT_STATUS`: `301` / `302` / `307` / `308` のみ許可。それ以外は起動エラー。
- `PORT`: 1–65535 の整数。

---

## 3. エンドポイント

| メソッド | パス | レスポンス | 説明 |
|----------|------|-----------|------|
| `GET` | `/health` | `200 OK` + `{"status":"ok"}` | Liveness probe |
| `GET` | `/ready` | `200 OK` + `{"status":"ready"}` | Readiness probe |
| `GET` | `/metrics` | Prometheus テキスト形式 | メトリクス |
| `ANY` | `/*` | 設定されたステータスコード + `Location` ヘッダ | 全リクエストをリダイレクト |

### リダイレクトのルール
- リクエストパス・クエリパラメータを **そのまま保持** して転送先 URL に結合する。
- `REDIRECT_DESTINATION` にパスが含まれている場合も適切に解決する（末尾スラッシュの正規化を行う）。

### ヘルスチェック
- `/health`: 常に 200。依存サービスがないため常に healthy。
- `/ready`: 常に 200。起動完了と同時に ready。

### メトリクス
Prometheus 形式で `/metrics` に公開:
- `redirect_requests_total{method, status_code}` — リダイレクト総数 (Counter)
- `redirect_request_duration_seconds{method, status_code}` — リクエスト処理時間 (Histogram)

---

## 4. アーキテクチャ

```
main.go              # エントリポイント（サーバ起動、Graceful Shutdown）
config.go            # 設定のパース・検証
handler.go           # HTTP ハンドラ（redirect / health / metrics）
handler_test.go      # テスト
```

- 構造体: `redirectHandler`（destination URL + status code）
- テスト: テーブル駆動テスト + httptest

---

## 5. 技術スタック

| 項目 | 選択 |
|------|------|
| 言語 | Go (1.24+) |
| HTTP | `net/http` (標準ライブラリ) |
| ロギング | `log/slog` (構造化ログ) |
| メトリクス | `prometheus/client_golang` |
| テスト | `testing` + `net/http/httptest` |
| Lint | `golangci-lint` |
| コンテナ | Docker マルチステージビルド (distroless-static) |
| リリース | goreleaser + tagpr + GitHub Actions |
| レジストリ | `ghcr.io/whywaita/redirector` |

---

## 6. Docker

```dockerfile
# Stage 1: Build
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /redirector .

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /redirector /redirector
EXPOSE 8080
ENTRYPOINT ["/redirector"]
```

- 非 root ユーザ (`nonroot`) で実行。
- イメージサイズは 10MB 以下を目標。

---

## 7. リリースフロー

GitHub Actions による自動化。`main` ブランチへの push で:

1. **tagpr**: 前回リリースからのコミットを解析し、バージョンタグを自動生成
2. **goreleaser**: タグが生成された場合のみ発火
   - クロスコンパイルバイナリ (linux/darwin, amd64/arm64) をビルド
   - Docker イメージをビルドし `ghcr.io/whywaita/redirector:latest` + `:vX.Y.Z` でプッシュ
3. GitHub Release にバイナリを添付

---

## 8. CI

| ワークフロー | トリガー | 内容 |
|-------------|---------|------|
| `test.yaml` | push/PR to main | `go build` + `go test -race` + `go vet` |
| `lint.yaml` | push/PR to main | `golangci-lint` |
| `release.yaml` | push to main | tagpr → goreleaser |

---

## 9. ファイル構成

```
.
├── main.go
├── handler.go
├── handler_test.go
├── config.go
├── go.mod
├── go.sum
├── Dockerfile
├── .goreleaser.yaml
├── .github/
│   └── workflows/
│       ├── test.yaml
│       ├── lint.yaml
│       └── release.yaml
├── README.md
└── SPEC.md
```
