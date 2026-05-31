# Asset Risk System

一个用 Go 标准库实现的网络资产风险管理系统原型，包含资产、域名/子域名、IP 端口、组件证明和风险 HTTP 证据。

## 运行

```bash
go run ./cmd/asset-risk-server -addr :8080 -data data/assets.json
```

环境变量也可配置：

```bash
ASSET_RISK_ADDR=:8080 ASSET_RISK_DATA=data/assets.json go run ./cmd/asset-risk-server
```

## 创建资产

```bash
curl -s http://127.0.0.1:8080/assets \
  -H 'Content-Type: application/json' \
  -d '{
    "primary_domain": "example.com",
    "ips": [{
      "address": "203.0.113.10",
      "ports": [{
        "port": 443,
        "protocol": "tcp",
        "service": "https",
        "banner": "nginx/1.24",
        "tls": true
      }]
    }],
    "domains": [{
      "name": "api.example.com",
      "kind": "subdomain"
    }],
    "components": [{
      "name": "nginx",
      "version": "1.24",
      "proof_url": "https://example.com/",
      "response_content": "HTTP/1.1 200 OK\r\nServer: nginx/1.24\r\n\r\n"
    }]
  }'
```

如果资产没有域名，可以省略 `primary_domain`，系统会使用第一个 IP 作为资产主标识。

## 追加风险

把 `{asset_id}` 替换为创建资产后返回的 `id`：

```bash
curl -s "http://127.0.0.1:8080/assets/{asset_id}/domains/api.example.com/risks" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "admin console exposed",
    "severity": "high",
    "url": "https://api.example.com/admin",
    "request": "GET /admin HTTP/1.1\r\nHost: api.example.com\r\n\r\n",
    "response": "HTTP/1.1 200 OK\r\n\r\nadmin"
  }'
```

## 常用查询

```bash
curl -s http://127.0.0.1:8080/assets
curl -s 'http://127.0.0.1:8080/assets?q=example'
curl -s 'http://127.0.0.1:8080/assets?severity=high'
curl -s http://127.0.0.1:8080/summary
curl -s "http://127.0.0.1:8080/assets/{asset_id}/risks?severity=high"
```

## 设计文档

见 [docs/design.md](docs/design.md)。

## 测试

```bash
go test ./...
```
