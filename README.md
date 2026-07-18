# subai — AI-Managed Subscription Converter & Rule Engine

高性能、轻量级的代理订阅转换工具，专为 AI 管理设计。

**核心特性：**

- 🚀 **轻量高性能** — 单二进制 ~6MB，启动 < 100ms，零运行时依赖
- 🤖 **AI 原生交互** — CLI 设计，AI 可直接读写配置、执行命令
- 📦 **规则仓库集成** — 内置 107 条规则集索引，支持搜索/添加/管理
- 🎯 **多场景 Profile** — 多配置集管理，一键切换场景
- 🔧 **规则编排** — 补丁插入、顺序调整，无需修改原始模板
- 🔄 **自动化运维** — 定时刷新 + Webhook 通知客户端热重载
- 🔐 **三阶段安全** — `validate` → `dry-run` → `convert`
- 🔑 **Token 鉴权** — HTTP 服务支持 `?token=` 参数认证

---

## Quick Start

```bash
# 1. 添加订阅源
subai source add my-airport "https://example.com/sub?token=xxx"

# 2. 搜索并添加规则
subai rule search netflix
subai rule add blackmatrix7/Netflix

# 3. 转换
subai convert

# 4. 启动 HTTP 服务（自动刷新 + 客户端通知）
subai serve --listen :8080 --token your-token \
  --auto-refresh --refresh-interval 6h \
  --webhook-url http://127.0.0.1:9090/configs?force=true
```

---

## 安装

**前置依赖：** 无（单二进制，零运行时依赖）

```bash
# 从源码构建
git clone https://github.com/Li-Qifeng/subai.git
cd subai
go build -o subai ./cmd/subai/
```

**面板登录功能**（`subai login`）需要 Python 3.8+：

```bash
pip3 install cloudscraper
```

---

## CLI 命令

### 订阅源管理

| 命令 | 说明 |
|------|------|
| `subai source add <name> <url>` | 添加订阅源 |
| `subai source list` | 列出所有源 |
| `subai source remove <name>` | 删除订阅源 |

### 规则仓库管理（Phase 1）

```bash
# 列出所有可用规则集（支持过滤）
subai rule list                         # 全部
subai rule list --repo blackmatrix7     # 按仓库过滤
subai rule list --category Proxy        # 按分类过滤
subai rule list --behavior ipcidr       # 按行为过滤

# 搜索规则集
subai rule search netflix
subai rule search openai --repo blackmatrix7

# 管理配置中的规则
subai rule add blackmatrix7/Netflix                     # 按仓库名引用
subai rule add "ACL4SSR/ACL4SSR/Clash/BanAD.list"      # 按路径引用
subai rule remove netflix                               # 删除规则
```

### 多场景 Profile（Phase 2）

```bash
# 创建场景
subai profile create mobile --template basic
subai profile create home --template acl4ssr_full

# 切换场景
subai profile switch mobile

# 查看所有场景
subai profile list

# 删除场景
subai profile delete mobile

# 临时使用指定场景
subai convert --profile home
```

### 规则编排（Phase 3）

```bash
# 在指定位置插入补丁规则
subai rule patch add "DOMAIN-SUFFIX,example.com,Proxy" \
  --insert-before MATCH --id my-rule

subai rule patch add "GEOIP,CN,DIRECT" \
  --insert-after "国内直连" --id geoip

# 查看已应用的补丁
subai rule patch list

# 删除补丁
subai rule patch remove my-rule

# 清空所有补丁
subai rule patch clear

# 调整规则顺序
subai rule order netflix --move-up
subai rule order openai --move-down
subai rule order google --to 3
```

### 转换输出

```bash
# 基础转换
subai convert

# 指定场景和输出格式
subai convert --profile mobile --target base64

# 预览（不输出文件）
subai dry-run

# 验证配置
subai validate
```

### HTTP 服务（Phase 4）

```bash
subai serve \
  --listen :8080 \
  --token your-api-token \
  --auto-refresh \
  --refresh-interval 6h \
  --output /path/to/output.yaml \
  --webhook-url http://127.0.0.1:9090/configs?force=true
```

**端点：**

| 路径 | 说明 |
|------|------|
| `GET /sub` | 获取完整 Clash 配置 |
| `GET /sub?profile=mobile` | 指定场景 |
| `GET /sub?token=xxx` | 带 Token 认证 |
| `GET /health` | 健康检查 |
| `POST /refresh` | 手动触发刷新 |

### 自动化面板登录

```bash
subai login my-airport \
  --method v2board \
  --url https://panel.example.org \
  --email user@example.com \
  --password "your-password"
```

支持 V2Board 面板的自动登录，含 Cloudflare 绕过。

---

## 配置参考

完整 `subai.yaml` 示例：

```yaml
sources:
  - name: my-airport
    url: "https://example.com/sub?token=xxx"
    user-agent: "ClashMeta/1.18"
    login:
      method: v2board
      url: "https://panel.example.org"
      email: "user@example.com"
      password: "your-password"

rules:
  include: ["🇭🇰|HK|Hong", "🇯🇵|JP|Japan"]
  exclude: ["过期", "剩余流量"]

output:
  target: clash              # clash, base64, mixed
  template: acl4ssr_full     # 内置模板
  fetch_rules: false         # true=内联展开, false=RULE-SET引用
  rule_providers: true       # 生成 rule-providers 段
  provider_interval: 86400   # 规则更新间隔（秒）
  rule_sets:
    - group: "🚀 代理"
      url: "https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script/master/rule/Clash/Proxy/Proxy.yaml"
  rule_patches:
    - id: geoip-cn
      position: "before:MATCH"
      rule: "GEOIP,CN,DIRECT"

server:
  listen: ":8080"
  token: "your-api-token"
  auto_refresh: true
  refresh_interval: 6h
  webhook_urls:
    - "http://127.0.0.1:9090/configs?force=true"

current_profile: default
profiles:
  mobile:
    sources:
      - name: my-airport
    output:
      target: clash
      template: basic
  home:
    sources:
      - name: my-airport
    output:
      target: clash
      template: acl4ssr_full
```

---

## 内置模板

| 模板名 | 策略组数 | 说明 |
|--------|----------|------|
| `basic` | 4 | 简单 select/url-test/fallback |
| `acl4ssr_full` | 13 | 完整规则（Netflix/YouTube/Google/Steam/AI等） |
| `acl4ssr_lite` | 5 | 精简版，仅选择/直连/拦截 |
| `loyalsoldier` | 6 | 精选规则，直连/代理/拦截覆盖全面 |

```bash
subai template list
subai template sync    # 同步远程模板
```

---

## 规则仓库索引

内置 3 个主流规则仓库，共 107 条规则集：

| 仓库 | 规则集数 | 来源 |
|------|----------|------|
| blackmatrix7 | 50+ | [ios_rule_script](https://github.com/blackmatrix7/ios_rule_script) |
| ACL4SSR | 30+ | [ACL4SSR](https://github.com/ACL4SSR/ACL4SSR) |
| Loyalsoldier | 20+ | [clash-rules](https://github.com/Loyalsoldier/clash-rules) |

---

## 三种规则输出模式

| 模式 | 配置 | 输出 |
|------|------|------|
| 内联展开 | `fetch_rules: true` | 规则内容直接嵌入 YAML，兼容所有客户端 |
| rule-providers | `rule_providers: true` | 生成 `rule-providers` 段，Clash Meta/Mihomo 自动拉取 |
| RULE-SET 引用 | 两者都 false | 传统 `RULE-SET,url,group` 引用 |

---

## 架构

```
┌─────────────────────────────────────┐
│         CLI Interface               │
│  convert / validate / login / serve │
│  rule / profile / patch             │
├─────────────────────────────────────┤
│       Config Manager (YAML)         │
│  sources / profiles / rules / patch │
├─────────────────────────────────────┤
│         Core Engine                 │
│  fetch → parse → filter → render    │
├─────────────────────────────────────┤
│    Rule Repository Index            │
│  blackmatrix7 / ACL4SSR / Loyalsoldier │
├─────────────────────────────────────┤
│    HTTP Server (可选)                │
│  /sub /health /refresh              │
│  auto-refresh / webhook             │
└─────────────────────────────────────┘
```

---

## 支持的格式

**输入：** SS, SSR, VMess, VLESS, Trojan, Hysteria2, TUIC, SOCKS5, HTTP, Clash YAML, Base64 Subscription

**输出：** Clash YAML (with proxy-groups + rules), Base64 URI List

---

## 客户端更新通知

两种模式：

1. **HTTP 订阅（推荐）** — 运行 `subai serve`，客户端指向 `http://host:8080/sub?token=xxx`，支持自动刷新

2. **Webhook 推送** — 配合 `--auto-refresh`，刷新完成后自动通知客户端热重载
   - 支持 Clash/Mihomo REST API（`PUT /configs?force=true`）
   - 支持任意自定义 URL（Telegram、Bark 等）

---

## 安全

- `validate` + `dry-run` + `convert` 三阶段，防止错误配置生效
- HTTP 服务支持 `?token=` 参数认证
- 凭证在 `source list` 输出中自动隐藏
- 建议限制配置文件的文件权限

---

## 性能

| 场景 | 指标 |
|------|------|
| 二进制大小 | ~6 MB (静态编译) |
| 启动时间 | < 100ms |
| 内存占用 | ~10-15 MB (idle) |
| 单次转换 | < 1s (20 节点) |
| 登录耗时 | ~10-30s (含 CF 挑战) |