# subai — AI-Managed Subscription Converter

高性能、轻量级的代理订阅转换工具，专为 AI 管理设计。

**核心特性：**
- 🚀 **轻量高性能** — 单二进制 ~6MB，启动 < 100ms，零运行时依赖
- 🤖 **AI 原生交互** — CLI 设计，AI 可直接读写配置、执行命令
- 🔐 **三阶段安全** — `validate` → `dry-run` → `convert`，防止出错配置
- 🎯 **确定性子规则** — 纯子串/正则匹配，不引入 JS 脚本
- 🔄 **自动化面板登录** — 支持 V2Board 面板自动登录，获取限时订阅链接
- 📡 **客户端通知** — 内置 HTTP 服务，客户端可订阅实时更新

## Quick Start

```bash
# 1. 自动登录机场面板获取订阅链接
subai login my-airport \
  --url https://www.example.org \
  --email user@example.com \
  --password "your-password"

# 2. 验证配置
subai validate -c subai.yaml

# 3. 转换
subai convert -c subai.yaml -o clash.yaml

# 4. 启动 HTTP 服务（客户端自动更新）
subai serve -c subai.yaml --listen ":8080"
# 客户端订阅地址: http://your-host:8080/sub?target=clash
```

## 安装

### 前置依赖

**运行转换功能** — 无需额外依赖，单二进制即可。

**面板登录功能**（`subai login`）— 需要 Python 3.8+ 和 cloudscraper：
```bash
pip3 install cloudscraper
```

### 下载二进制

```bash
# 从 Releases 下载
wget https://github.com/user/subai/releases/latest/download/subai-linux-amd64
chmod +x subai-linux-amd64
sudo mv subai-linux-amd64 /usr/local/bin/subai
```

## CLI Commands

| Command | Description | AI Usage |
|---------|-------------|----------|
| `convert` | Convert subscriptions to target format | Primary output command |
| `validate` | Validate config + rules | Pre-flight check |
| `dry-run` | Preview without writing output | Verification step |
| `source add/list/remove` | Manage subscription sources | Configure inputs |
| **`login`** | **Auto-login to panel, get subscribe URL** | **One-click setup** |
| `serve` | HTTP server for client subscriptions | Client delivery |
| `version` | Show version | - |

### 自动化面板登录 (`subai login`)

```bash
# 登录 V2Board 面板，自动获取订阅链接并写入配置
subai login my-airport \
  --method v2board \
  --url https://www.xfltd.org \
  --email user@example.com \
  --password "your-password"

# 输出示例:
#   🔐 Logging into https://www.xfltd.org as user@example.com...
#   ✅ Login successful!
#   📧 Email: user@example.com
#   📦 Plan: 250G-不限时长
#   📊 Traffic: 42.90 GB / 250.00 GB (17.2%)
#   🔗 Subscribe URL: https://get.cctvclient.cn/...
#   📝 Added source "my-airport" to config
#   ✅ Config saved to subai.yaml
```

登录成功后，`subai.yaml` 会自动写入：

```yaml
sources:
  - name: my-airport
    url: "https://get.cctvclient.cn/...?token=xxx"
    user-agent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"
    login:
      method: v2board
      url: "https://www.xfltd.org"
      email: "user@example.com"
      password: "your-password"
```

> **注意**：`login` 配置保存了面板登录凭证，可用于后续自动刷新订阅链接。`subai` 会**隐藏**密码和邮箱在 `source list` 输出中。

### 内联转换

```bash
# 直接代理 URI
subai convert -t clash "ss://YWVzLTI1Ni1nY206cGFzc3dk@1.1.1.1:443#node"

# 订阅 URL
subai convert -t base64 "https://example.com/sub?token=xxx"
```

## 支持的格式

**Input:** SS, SSR, VMess, VLESS, Trojan, Hysteria2, TUIC, SSD, SOCKS5, HTTP, Clash YAML, Base64 Subscription

**Output:** Clash YAML (with proxy-groups + rules), Base64 URI List

## 配置参考

```yaml
sources:
  - name: "source-name"
    url: "https://..."
    cookie: "session=xxx"                # 可选，用于限时订阅
    user-agent: "ClashMeta/1.18"          # 可选，覆盖默认 UA
    refresh-cron: "0 */6 * * *"           # 可选，自动刷新周期
    login:                                # 可选，面板登录配置
      method: "v2board"                   # 登录方法 (v2board)
      url: "https://panel.example.org"    # 面板地址
      email: "user@example.com"           # 登录邮箱
      password: "your-password"           # 登录密码

rules:
  include: ["🇭🇰|HK|Hong", "🇯🇵|JP|Japan"]
  exclude: ["过期", "剩余流量"]

output:
  target: "clash"        # clash, base64, mixed
  pretty: true
  # 模板配置（可选）
  template: "acl4ssr_mini"              # 内置模板名
  proxy-groups:                         # 自定义代理组（覆盖模板）
    - name: "Proxy"
      type: "select"
      filter: "^(?!.*(过期|官网)).*$"
  rule-sets:                            # 自定义规则集
    - group: "Proxy"
      url: "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/ProxyGFWlist.list"

server:
  enabled: true
  listen: ":8080"
  token: "your-api-token"
```

## 面板登录支持

### V2Board

自动登录 V2Board 面板的流程：

1. **Cloudflare 绕过** — 通过 Python cloudscraper 自动处理 CF JS 挑战
2. **GE-UA 挑战** — 支持自定义 GE-UA 安全验证的自动求解
3. **登录 API** — `POST /api/v1/passport/auth/login`
4. **获取订阅** — 自动获取用户信息、套餐、流量、订阅链接
5. **配置持久化** — 保存订阅 URL 和 User-Agent 到配置文件

### 支持的机场面板

| 面板 | 方法名 | 登录端点 | 状态 |
|------|--------|----------|------|
| V2Board | `v2board` | `/api/v1/passport/auth/login` | ✅ 已验证 |

## 架构设计

```
┌─────────────────────────────────────┐
│         CLI Interface               │  ← AI 主交互层
│  convert / validate / login / serve │
├─────────────────────────────────────┤
│       Config Manager (YAML)         │  ← AI 读写配置，确定性语法
│  sources / rules / templates / login│
├─────────────────────────────────────┤
│         Core Engine                 │  ← 高性能转换核心
│  fetch → parse → filter → render    │
├─────────────────────────────────────┤
│  Login Engine (Python cloudscraper) │  ← 面板登录自动化
│  CF bypass → V2Board login → sub   │
├─────────────────────────────────────┤
│    HTTP Server (可选)                │  ← 为客户端提供订阅链接
│  /sub /health /webhook              │
└─────────────────────────────────────┘
```

## 客户端更新通知

两种模式：

1. **HTTP 订阅（推荐）** — 运行 `subai serve`，客户端指向 `http://host:8080/sub?target=clash`，客户端自动定期拉取更新。

2. **文件输出** — 运行 `subai convert -o /path/to/config.yaml`，通过任意静态文件服务器（nginx、S3 等）提供服务。

## 安全

- `validate` + `dry-run` + `convert` 三阶段，防止错误配置生效
- 凭证在 `source list` 输出中自动隐藏
- 密码以明文存储在配置文件中 — 建议限制配置文件的文件权限
- `login` 配置仅用于自动刷新，不影响核心转换功能

## 性能

| 场景 | 指标 |
|------|------|
| 二进制大小 | 6.3 MB (静态编译, UPX 可压缩至 ~2MB) |
| 启动时间 | < 100ms |
| 内存占用 | ~10-15 MB (idle) |
| 单次转换 | < 1s (20 节点, 含规则过滤) |
| 登录耗时 | ~10-30s (含 CF 挑战) |