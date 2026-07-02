# subai — AI-Managed Subscription Converter

高性能、轻量级的代理订阅转换工具，专为 AI 管理设计。

## Design Philosophy

- **CLI-first**: 所有操作通过命令行完成，无 Web UI 依赖
- **Config as Code**: YAML 配置，AI 可直接读写、版本控制
- **Deterministic Rules**: 规则用确定性子串/正则，不引入 JS 脚本
- **Safety First**: `validate` → `dry-run` → `convert` 三阶段，防止出错配置
- **Lightweight**: 单二进制 ~6MB，启动 < 100ms，零运行时依赖

## Quick Start

```bash
# 1. Create config
cat > subai.yaml << 'EOF'
sources:
  - name: "my-airport"
    url: "https://example.com/sub?token=xxx"
    cookie: ""  # optional, for time-limited subscriptions
rules:
  include:
    - "🇭🇰|HK|Hong"
    - "🇯🇵|JP|Japan"
  exclude:
    - "过期"
output:
  target: "clash"
EOF

# 2. Validate config
subai validate -c subai.yaml

# 3. Preview without writing
subai dry-run -c subai.yaml

# 4. Convert
subai convert -c subai.yaml -o clash.yaml

# 5. Serve (for client auto-update)
subai serve -c subai.yaml --listen ":8080"
# Client subscribes to: http://your-host:8080/sub?target=clash
```

## CLI Commands

| Command | Description | AI Usage |
|---------|-------------|----------|
| `convert` | Convert subscriptions to target format | Primary output command |
| `validate` | Validate config + rules | Pre-flight check |
| `dry-run` | Preview without writing output | Verification step |
| `source add/list/remove` | Manage subscription sources | Configure inputs |
| `serve` | HTTP server for client subscriptions | Client delivery |
| `version` | Show version | - |

### Inline Conversion

```bash
# Direct proxy URI
subai convert -t clash "ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443#node"

# Subscription URL
subai convert -t base64 "https://example.com/sub?token=xxx"
```

## Supported Formats

**Input:** SS, SSR, VMess, VLESS, Trojan, Hysteria2, TUIC, SSD, SOCKS5, HTTP, Clash YAML, Base64 Subscription

**Output:** Clash YAML (with proxy-groups + rules), Base64 URI List

## Config Reference

```yaml
sources:
  - name: "source-name"
    url: "https://..."
    cookie: "session=xxx"     # For time-limited airport subscriptions
    user-agent: "ClashMeta/1.18"

rules:
  include: ["pattern1", "pattern2"]  # Regex/substring match on name
  exclude: ["pattern1"]              # Applied after include

output:
  target: "clash"        # clash, base64, mixed
  pretty: true

server:
  enabled: true
  listen: ":8080"
  token: "your-api-token"
```

## Client Update Notification

Two modes for notifying clients of config updates:

1. **HTTP Subscription** (recommended): Run `subai serve`, point your client to `http://host:8080/sub?target=clash`. Client auto-pulls on its update interval.

2. **File-based**: Run `subai convert -o /path/to/config.yaml`, serve via any static file server (nginx, S3, etc.).