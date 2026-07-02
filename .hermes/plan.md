# subai — AI-Managed Subscription Converter

## Overview
A high-performance, lightweight subscription conversion tool designed for AI management (CLI-only, no UI).

## Architecture
- **Language:** Go (single static binary, ~15MB, startup < 100ms)
- **Config:** YAML — AI-readable/writable, deterministic rules
- **Interface:** CLI (`subai`) + optional lightweight HTTP server (`subai serve`)

## Phase 1 — Core Engine (MVP)
| Component | Description | Files |
|-----------|-------------|-------|
| Config | YAML config with sources, rules, output settings | `internal/config/config.go` |
| Fetcher | HTTP fetch with cookie/session support | `internal/fetcher/fetcher.go` |
| Parser | Parse SS/VMess/Trojan/VLESS/Clash YAML/base64 | `internal/parser/*.go` |
| Converter | Convert to Clash YAML / base64 output | `internal/converter/converter.go` |
| Rule | Deterministic include/exclude rule engine | `internal/rule/rule.go` |
| CLI | Cobra-based CLI with validate/dry-run | `cmd/subai/main.go` |
| Server | Lightweight HTTP for client subscription | `internal/server/server.go` |

## Phase 2 — Extended Formats & Merge
- Surge/Quantumult X/Loon/sing-box support
- Subscription merging with dedup
- Template system (proxy groups, routing rules)

## Phase 3 — AI Automation
- Cookie/session auto-refresh for time-limited subscriptions
- Webhook notification on config change
- Atomic config apply with rollback

## Success Criteria
- `subai convert -o out.yaml` produces valid Clash YAML
- `subai validate` catches rule errors before apply
- `subai serve` provides subscription URL for clients
- Single binary, no runtime deps
