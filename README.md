# redirector

> 受け取った HTTP リクエストを指定ドメインにリダイレクトするだけの極薄アプリケーション

[![test](https://github.com/whywaita/redirector/actions/workflows/test.yaml/badge.svg)](https://github.com/whywaita/redirector/actions/workflows/test.yaml)
[![lint](https://github.com/whywaita/redirector/actions/workflows/lint.yaml/badge.svg)](https://github.com/whywaita/redirector/actions/workflows/lint.yaml)

## Quick Start

### バイナリ

```bash
# 環境変数で設定
export REDIRECT_DESTINATION=https://example.com
export REDIRECT_STATUS=301
./redirector
```

```bash
# CLI フラグで設定
./redirector --destination https://example.com --status 301 --port 8080
```

### Docker

```bash
docker run -p 8080:8080 \
  -e REDIRECT_DESTINATION=https://example.com \
  -e REDIRECT_STATUS=301 \
  ghcr.io/whywaita/redirector:latest
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redirector
spec:
  replicas: 2
  selector:
    matchLabels:
      app: redirector
  template:
    metadata:
      labels:
        app: redirector
    spec:
      containers:
        - name: redirector
          image: ghcr.io/whywaita/redirector:latest
          ports:
            - containerPort: 8080
          env:
            - name: REDIRECT_DESTINATION
              value: "https://example.com"
            - name: REDIRECT_STATUS
              value: "301"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
```

## 設定

| 環境変数 | CLI フラグ | デフォルト | 説明 |
|----------|-----------|------------|------|
| `REDIRECT_DESTINATION` | `--destination` / `-d` | **必須** | リダイレクト先ベース URL |
| `REDIRECT_STATUS` | `--status` / `-s` | `302` | `301`, `302`, `307`, `308` |
| `PORT` | `--port` / `-p` | `8080` | 待受ポート |

CLI フラグが環境変数より優先されます。

## エンドポイント

| パス | 説明 |
|------|------|
| `/health` | Liveness probe (`200 OK`) |
| `/ready` | Readiness probe (`200 OK`) |
| `/metrics` | Prometheus メトリクス |
| `/*` | リダイレクト |

## メトリクス

Prometheus メトリクスが `/metrics` で公開されます。

- `redirect_requests_total{method, status_code}` — リダイレクト総数
- `redirect_request_duration_seconds{method, status_code}` — リクエスト処理時間

## リダイレクトの挙動

リクエストパスとクエリパラメータはそのまま転送先に結合されます。

```
GET /foo/bar?q=hello + REDIRECT_DESTINATION=https://example.com
  → Location: https://example.com/foo/bar?q=hello
```

`REDIRECT_DESTINATION` にパスが含まれる場合も適切に解決されます。

```
REDIRECT_DESTINATION=https://example.com/api
GET /v1/users
  → Location: https://example.com/api/v1/users
```

## ライセンス

MIT
