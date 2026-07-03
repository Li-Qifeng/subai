#!/usr/bin/env python3
"""Generate all ACL4SSR variant template YAML files + index.json."""
import json, os, yaml

TEMPLATES_DIR = os.path.dirname(os.path.abspath(__file__))

# ── Rule set URLs (ACL4SSR master) ──────────────────────────
CDN = "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash"
R = lambda name: f"{CDN}/{name}.list"

BAN = [
    R("BanAD"), R("BanProgramAD"), R("BanEasyList"), R("BanEasyListChina"),
]
BAN_LITE = [R("BanAD"), R("BanEasyList")]
PROXY = [R("ProxyGFWlist"), R("ProxyLite")]
DIRECT_CHINA = [R("ChinaDomain"), R("ChinaIp"), R("LocalAreaNetwork")]
STREAMING = [
    (R("Netflix"), "📺 Netflix"),
    (R("YouTube"), "📽 YouTube"),
    (R("Google"), "🌐 谷歌服务"),
    (R("Apple"), "🍎 苹果服务"),
    (R("Twitter"), "🐦 Twitter"),
    (R("Telegram"), "📲 Telegram"),
    (R("Steam"), "🎮 游戏平台"),
    (R("OpenAi"), "🤖 OpenAI"),
    (R("Microsoft"), "🍎 苹果服务"),
]
STREAMING_FULL = [
    (R("Netflix"), "📺 Netflix"),
    (R("YouTube"), "📽 YouTube"),
    (R("Google"), "🌐 谷歌服务"),
    (R("Apple"), "🍎 苹果服务"),
    (R("Twitter"), "🐦 Twitter"),
    (R("Telegram"), "📲 Telegram"),
    (R("Steam"), "🎮 游戏平台"),
    (R("OpenAi"), "🤖 OpenAI"),
    (R("Microsoft"), "🍎 苹果服务"),
    (R("Netflix"), "📺 Netflix"),
]

# ── Proxy group definitions ─────────────────────────────────
URL = "http://www.gstatic.com/generate_204"
INT = 300

def P(name, type, filter=None, proxies=None, url=URL, interval=INT):
    g = {"name": name, "type": type}
    if proxies: g["proxies"] = proxies
    if filter:  g["filter"] = filter
    if type in ("url-test","fallback","load-balance"):
        g["url"] = url
        g["interval"] = interval
    return g

# ── Template variants ───────────────────────────────────────

def make_acl4ssr_standard(extra_rules=None):
    """12 proxy groups + MATCH, used by standard/full/google/netflix variants"""
    return {
        "proxy_groups": [
            P("🚀 节点选择", "select", proxies=["*"]),
            P("♻️ 自动选择", "url-test", proxies=["*"]),
            P("📺 Netflix", "select", filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
            P("📽 YouTube", "select", filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
            P("🌐 谷歌服务", "select", filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
            P("🍎 苹果服务", "select", filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
            P("🐦 Twitter", "select", filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
            P("📲 Telegram", "select", filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
            P("🎮 游戏平台", "select", filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
            P("🤖 OpenAI", "select", filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
            P("🇨🇳 国内流量", "select", proxies=["DIRECT","🚀 节点选择"]),
            P("🚫 广告拦截", "select", proxies=["REJECT","DIRECT"]),
            P("🐟 漏网之鱼", "select", proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
        ],
        "rule_sets": (extra_rules or []) + [
            *[{"url": u, "group": "🚫 广告拦截"} for u in BAN],
            *[{"url": u, "group": g} for u, g in STREAMING],
            *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
            {"rule": "MATCH,🐟 漏网之鱼"},
        ],
    }

def make_standard_groups():
    """Same groups as standard but rule_sets differ per variant"""
    return [
        P("🚀 节点选择", "select", proxies=["*"]),
        P("♻️ 自动选择", "url-test", proxies=["*"]),
        P("📺 Netflix", "select", filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube", "select", filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务", "select", filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务", "select", filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter", "select", filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram", "select", filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台", "select", filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI", "select", filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量", "select", proxies=["DIRECT","🚀 节点选择"]),
        P("🚫 广告拦截", "select", proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼", "select", proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ]

TEMPLATES = {}

# ── 1. ACL4SSR_Online ──
TEMPLATES["acl4ssr_online"] = {
    "name": "ACL4SSR Online",
    "description": "ACL4SSR Online 默认版，分组比较全",
    "category": "ACL4SSR",
    "proxy_groups": make_standard_groups(),
    "rule_sets": [
        *[{"url": u, "group": "🚫 广告拦截"} for u in BAN],
        *[{"url": u, "group": g} for u, g in STREAMING],
        *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── 2. ACL4SSR_Online_AdblockPlus ──
TEMPLATES["acl4ssr_online_adblockplus"] = {
    "name": "ACL4SSR Online AdblockPlus",
    "description": "ACL4SSR Online 更多去广告",
    "category": "ACL4SSR",
    "proxy_groups": make_standard_groups(),
    "rule_sets": [
        *[{"url": u, "group": "🚫 广告拦截"} for u in BAN],
        {"url": R("BanEasyPrivacy"), "group": "🚫 广告拦截"},
        *[{"url": u, "group": g} for u, g in STREAMING],
        *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── 3. ACL4SSR_Online_NoAuto ──
TEMPLATES["acl4ssr_online_noauto"] = {
    "name": "ACL4SSR Online NoAuto",
    "description": "ACL4SSR Online 无自动测速",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("📺 Netflix","select",filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube","select",filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务","select",filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务","select",filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter","select",filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram","select",filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台","select",filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI","select",filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🚫 广告拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","DIRECT"]),
    ],
    "rule_sets": [
        *[{"url": u, "group": "🚫 广告拦截"} for u in BAN],
        *[{"url": u, "group": g} for u, g in STREAMING],
        *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── 4. ACL4SSR_Online_NoReject ──
TEMPLATES["acl4ssr_online_noreject"] = {
    "name": "ACL4SSR Online NoReject",
    "description": "ACL4SSR Online 无广告拦截规则",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("📺 Netflix","select",filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube","select",filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务","select",filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务","select",filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter","select",filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram","select",filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台","select",filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI","select",filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": [
        *[{"url": u, "group": "🚀 节点选择"} for u in PROXY],
        *[{"url": u, "group": g} for u, g in STREAMING],
        *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── 5. ACL4SSR_Online_Mini ──
TEMPLATES["acl4ssr_online_mini"] = {
    "name": "ACL4SSR Online Mini",
    "description": "ACL4SSR Online 精简版",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": [
        *[{"url": u, "group": "🛑 全球拦截"} for u in BAN_LITE],
        *[{"url": u, "group": "🚀 节点选择"} for u in PROXY],
        *[{"url": u, "group": "🎯 全球直连"} for u in DIRECT_CHINA],
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── 6. Mini variants ──
MINI_GROUPS = [
    P("🚀 节点选择","select",proxies=["*"]),
    P("♻️ 自动选择","url-test",proxies=["*"]),
    P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
    P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
    P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
]

MINI_RULES = lambda extra: [
    *[{"url": u, "group": "🛑 全球拦截"} for u in BAN_LITE],
    *extra,
    *[{"url": u, "group": "🎯 全球直连"} for u in DIRECT_CHINA],
    {"rule": "MATCH,🐟 漏网之鱼"},
]

TEMPLATES["acl4ssr_online_mini_adblockplus"] = {
    "name": "ACL4SSR Online Mini AdblockPlus",
    "description": "ACL4SSR Online 精简版 更多去广告",
    "category": "ACL4SSR",
    "proxy_groups": MINI_GROUPS,
    "rule_sets": MINI_RULES([
        {"url": R("BanEasyPrivacy"), "group": "🛑 全球拦截"},
        *[{"url": u, "group": "🚀 节点选择"} for u in PROXY],
    ]),
}

TEMPLATES["acl4ssr_online_mini_fallback"] = {
    "name": "ACL4SSR Online Mini Fallback",
    "description": "ACL4SSR Online 精简版 带故障转移",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 故障转移","fallback",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 故障转移","DIRECT"]),
    ],
    "rule_sets": MINI_RULES([*[{"url": u, "group": "🚀 节点选择"} for u in PROXY]]),
}

TEMPLATES["acl4ssr_online_mini_multimode"] = {
    "name": "ACL4SSR Online Mini MultiMode",
    "description": "ACL4SSR Online 精简版 自动测速/故障转移/负载均衡",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("🔁 故障转移","fallback",proxies=["*"]),
        P("⚖️ 负载均衡","load-balance",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": MINI_RULES([*[{"url": u, "group": "🚀 节点选择"} for u in PROXY]]),
}

TEMPLATES["acl4ssr_online_mini_multicountry"] = {
    "name": "ACL4SSR Online Mini MultiCountry",
    "description": "ACL4SSR Online 精简版 带港美日国家分组",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("🇭🇰 香港节点","select",filter="🇭🇰|HK|Hongkong|Hong Kong"),
        P("🇺🇸 美国节点","select",filter="🇺🇸|US|United States|美国"),
        P("🇯🇵 日本节点","select",filter="🇯🇵|JP|Japan|日本"),
        P("🇸🇬 新加坡节点","select",filter="🇸🇬|SG|Singapore|新加坡"),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": MINI_RULES([*[{"url": u, "group": "🚀 节点选择"} for u in PROXY]]),
}

TEMPLATES["acl4ssr_online_mini_noauto"] = {
    "name": "ACL4SSR Online Mini NoAuto",
    "description": "ACL4SSR Online 精简版 不带自动测速",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🛑 全球拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","DIRECT"]),
    ],
    "rule_sets": MINI_RULES([*[{"url": u, "group": "🚀 节点选择"} for u in PROXY]]),
}

# ── 11-16: Full variants ──
FULL_GROUPS = make_standard_groups()
FULL_RULES = lambda replaces: [
    *[{"url": u, "group": "🚫 广告拦截"} for u in BAN + [R("BanEasyPrivacy")]],
    *replaces,
    *[{"url": u, "group": g} for u, g in STREAMING],
    *[{"url": u, "group": "🇨🇳 国内流量"} for u in DIRECT_CHINA],
    {"rule": "MATCH,🐟 漏网之鱼"},
]

TEMPLATES["acl4ssr_online_full"] = {
    "name": "ACL4SSR Online Full",
    "description": "ACL4SSR Online 全分组 重度用户使用",
    "category": "ACL4SSR",
    "proxy_groups": FULL_GROUPS,
    "rule_sets": FULL_RULES([]),
}

TEMPLATES["acl4ssr_online_full_multimode"] = {
    "name": "ACL4SSR Online Full MultiMode",
    "description": "ACL4SSR Online 全分组 多模式 重度用户使用",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("🔁 故障转移","fallback",proxies=["*"]),
        P("⚖️ 负载均衡","load-balance",proxies=["*"]),
        P("📺 Netflix","select",filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube","select",filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务","select",filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务","select",filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter","select",filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram","select",filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台","select",filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI","select",filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🚫 广告拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","🔁 故障转移","⚖️ 负载均衡","DIRECT"]),
    ],
    "rule_sets": FULL_RULES([]),
}

TEMPLATES["acl4ssr_online_full_noauto"] = {
    "name": "ACL4SSR Online Full NoAuto",
    "description": "ACL4SSR Online 全分组 无自动测速 重度用户使用",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("📺 Netflix","select",filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube","select",filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务","select",filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务","select",filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter","select",filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram","select",filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台","select",filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI","select",filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🚫 广告拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","DIRECT"]),
    ],
    "rule_sets": FULL_RULES([]),
}

TEMPLATES["acl4ssr_online_full_adblockplus"] = {
    "name": "ACL4SSR Online Full AdblockPlus",
    "description": "ACL4SSR Online 全分组 更多去广告 重度用户使用",
    "category": "ACL4SSR",
    "proxy_groups": FULL_GROUPS,
    "rule_sets": FULL_RULES([{"url": R("BanEasyPrivacy2"), "group": "🚫 广告拦截"}]),
}

TEMPLATES["acl4ssr_online_full_netflix"] = {
    "name": "ACL4SSR Online Full Netflix",
    "description": "ACL4SSR Online 全分组 奈飞全量",
    "category": "ACL4SSR",
    "proxy_groups": FULL_GROUPS,
    "rule_sets": FULL_RULES([]) + [
        {"url": R("Netflix"), "group": "📺 Netflix"},
        {"url": R("NetflixIP"), "group": "📺 Netflix"},
    ],
}

TEMPLATES["acl4ssr_online_full_google"] = {
    "name": "ACL4SSR Online Full Google",
    "description": "ACL4SSR Online 全分组 谷歌细分",
    "category": "ACL4SSR",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("📺 Netflix","select",filter="🇭🇰|🇸🇬|NETFLIX|Netflix|nf|NF"),
        P("📽 YouTube","select",filter="YouTube|YT|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("🌐 谷歌服务","select",filter="Google|谷歌|🇭🇰|🇸🇬|🇺🇸"),
        P("🍎 苹果服务","select",filter="Apple|苹果|🇭🇰|🇸🇬|🇺🇸"),
        P("🐦 Twitter","select",filter="Twitter|推特|🇭🇰|🇯🇵|🇸🇬|🇺🇸"),
        P("📲 Telegram","select",filter="Telegram|电报|🇭🇰|🇸🇬|🇯🇵"),
        P("🎮 游戏平台","select",filter="Steam|Epic|Game|游戏|🇭🇰|🇯🇵|🇸🇬"),
        P("🤖 OpenAI","select",filter="OpenAI|ChatGPT|🇺🇸|🇯🇵"),
        P("🇨🇳 国内流量","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🚫 广告拦截","select",proxies=["REJECT","DIRECT"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": FULL_RULES([]) + [
        {"url": R("Google"), "group": "🌐 谷歌服务"},
        {"url": R("YouTube"), "group": "📽 YouTube"},
        {"url": R("GoogleFCM"), "group": "🌐 谷歌服务"},
    ],
}

# ── universal variants ──
TEMPLATES["universal_no_urltest"] = {
    "name": "Universal No-Urltest",
    "description": "通用模板 无自动测速",
    "category": "universal",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","DIRECT"]),
    ],
    "rule_sets": [
        {"rule": "GEOIP,CN,🎯 全球直连"},
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

TEMPLATES["universal_urltest"] = {
    "name": "Universal Urltest",
    "description": "通用模板 带自动测速",
    "category": "universal",
    "proxy_groups": [
        P("🚀 节点选择","select",proxies=["*"]),
        P("♻️ 自动选择","url-test",proxies=["*"]),
        P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
        P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
    ],
    "rule_sets": [
        {"rule": "GEOIP,CN,🎯 全球直连"},
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── Basic (always available as fallback) ──
TEMPLATES["basic"] = {
    "name": "Basic",
    "description": "基础模板，仅含节点选择/自动测速/Fallback/兜底",
    "category": "core",
    "proxy_groups": [
        P("Proxy","select",proxies=["*"]),
        P("Auto","url-test",proxies=["*"]),
        P("Fallback","fallback",proxies=["*"]),
        P("Final","select",proxies=["Proxy","Auto","DIRECT"]),
    ],
    "rule_sets": [
        {"rule": "MATCH,Final"},
    ],
}

# ── Special ──
TEMPLATES["special_basic"] = {
    "name": "Special Basic",
    "description": "仅GEOIP CN + Final",
    "category": "special",
    "proxy_groups": [
        P("Proxy","select",proxies=["*"]),
        P("🐟 漏网之鱼","select",proxies=["Proxy","DIRECT"]),
    ],
    "rule_sets": [
        {"rule": "GEOIP,CN,DIRECT"},
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

TEMPLATES["special_netease_unblock"] = {
    "name": "Special Netease Unblock",
    "description": "网易云音乐解锁 (仅规则, No-Urltest)",
    "category": "special",
    "proxy_groups": [
        P("Proxy","select",proxies=["*"]),
        P("🐟 漏网之鱼","select",proxies=["Proxy","DIRECT"]),
    ],
    "rule_sets": [
        {"rule": "DOMAIN-SUFFIX,163.com,Proxy"},
        {"rule": "DOMAIN-SUFFIX,126.net,Proxy"},
        {"rule": "DOMAIN-SUFFIX,netease.com,Proxy"},
        {"rule": "DOMAIN-SUFFIX,musicshy.com,Proxy"},
        {"rule": "GEOIP,CN,DIRECT"},
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}

# ── Loyalsoldier ──
LOYAL_GROUPS = [
    P("🚀 节点选择","select",proxies=["*"]),
    P("♻️ 自动选择","url-test",proxies=["*"]),
    P("🎯 全球直连","select",proxies=["DIRECT","🚀 节点选择"]),
    P("🍎 苹果服务","select",filter="Apple|iCloud|🇭🇰|🇸🇬|🇺🇸"),
    P("📲 Telegram","select",filter="Telegram|🇭🇰|🇸🇬|🇯🇵"),
    P("🐟 漏网之鱼","select",proxies=["🚀 节点选择","♻️ 自动选择","DIRECT"]),
]

LOYAL_CDN = "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release"

TEMPLATES["loyalsoldier"] = {
    "name": "Loyalsoldier",
    "description": "Loyalsoldier/clash-rules 风格 (27.4k stars)",
    "category": "community",
    "proxy_groups": LOYAL_GROUPS,
    "rule_sets": [
        {"url": f"{LOYAL_CDN}/direct.txt", "group": "🎯 全球直连"},
        {"url": f"{LOYAL_CDN}/private.txt", "group": "🎯 全球直连"},
        {"url": f"{LOYAL_CDN}/apple.txt", "group": "🍎 苹果服务"},
        {"url": f"{LOYAL_CDN}/proxy.txt", "group": "🚀 节点选择"},
        {"url": f"{LOYAL_CDN}/gfw.txt", "group": "🚀 节点选择"},
        {"url": f"{LOYAL_CDN}/telegramcidr.txt", "group": "📲 Telegram"},
        {"url": f"{LOYAL_CDN}/reject.txt", "group": "🎯 全球直连"},
        {"rule": "MATCH,🐟 漏网之鱼"},
    ],
}


# ════════════════════════════════════════════════
# Generate files
# ════════════════════════════════════════════════

def write_template(name, data):
    path = os.path.join(TEMPLATES_DIR, f"{name}.yaml")
    with open(path, "w", encoding="utf-8") as f:
        f.write("# Auto-generated by generate.py\n")
        f.write(f"# name: {data['name']}\n")
        f.write(f"# description: {data['description']}\n")
        f.write(f"# category: {data['category']}\n")
        f.write("---\n")
        yaml.dump({"proxy_groups": data["proxy_groups"], "rule_sets": data["rule_sets"]},
                   f, allow_unicode=True, default_flow_style=False, sort_keys=False)
    print(f"  ✓ {name}.yaml")

def build_index():
    index = []
    for name, data in sorted(TEMPLATES.items()):
        index.append({
            "name": name,
            "title": data["name"],
            "description": data["description"],
            "category": data["category"],
            "file": f"{name}.yaml",
        })
    path = os.path.join(TEMPLATES_DIR, "index.json")
    with open(path, "w", encoding="utf-8") as f:
        json.dump(index, f, ensure_ascii=False, indent=2)
    print(f"\n  ✓ index.json ({len(index)} templates)")
    return index

if __name__ == "__main__":
    print(f"Generating {len(TEMPLATES)} templates...\n")
    for name, data in sorted(TEMPLATES.items()):
        write_template(name, data)
    build_index()
    print("\nDone.")