#!/usr/bin/env python3
"""Convert subconverter ACL4SSR Local .ini templates to subai YAML format."""
import re, sys, yaml, urllib.request

def parse_ini_rule(rule):
    """Parse a ruleset line: ruleset=group,rule"""
    idx = rule.index(',')
    group = rule[:idx].strip()
    r = rule[idx+1:].strip()
    
    if r.startswith('[]FINAL'):
        return {'group': group, 'rule': 'MATCH,' + group}
    elif r.startswith('[]GEOSITE,'):
        geosite = r[len('[]GEOSITE,'):].replace(',no-resolve', '')
        return {'group': group, 'built_in': f'geosite:{geosite}'}
    elif r.startswith('[]GEOIP,'):
        geoip = r[len('[]GEOIP,'):].replace(',no-resolve', '')
        return {'group': group, 'built_in': f'geoip:{geoip}'}
    elif r.startswith('clash-domain:'):
        url = re.sub(r',\d+$', '', r[len('clash-domain:'):])
        return {'group': group, 'url': url}
    elif r.startswith('clash-classic:'):
        url = re.sub(r',\d+$', '', r[len('clash-classic:'):])
        return {'group': group, 'url': url}
    else:
        return {'group': group, 'rule': r}

def parse_ini_group(line):
    """Parse a custom_proxy_group line."""
    parts = line.split('`')
    name = parts[0].strip()
    gtype = parts[1].strip()
    rest = parts[2] if len(parts) > 2 else ''
    
    group = {'name': name, 'type': gtype}
    
    if gtype == 'select':
        proxies = []
        for item in rest.split('`'):
            item = item.strip()
            if item.startswith('[]'):
                proxies.append(item[2:])
        if proxies:
            group['proxies'] = proxies
    elif gtype in ('url-test', 'fallback', 'load-balance'):
        filter_part = rest
        url_part = parts[3] if len(parts) > 3 else ''
        interval_part = parts[4] if len(parts) > 4 else ''
        
        if filter_part.startswith('('):
            group['filter'] = filter_part
        elif filter_part == '.*':
            group['filter'] = '.*'
        
        if url_part:
            group['url'] = url_part
        if interval_part:
            try:
                group['interval'] = int(interval_part.split(',')[0])
            except ValueError:
                pass
    
    return group

def convert_ini_to_yaml(ini_content, template_name, description):
    """Convert .ini content to subai YAML."""
    rules = []
    groups = []
    
    for line in ini_content.split('\n'):
        line = line.strip()
        if not line or line.startswith(';'):
            continue
        if line.startswith('[custom]'):
            continue
        
        if line.startswith('ruleset='):
            rule_str = line[len('ruleset='):]
            try:
                rules.append(parse_ini_rule(rule_str))
            except Exception as e:
                print(f"  ⚠️  Skip rule: {rule_str[:60]} - {e}", file=sys.stderr)
        
        if line.startswith('custom_proxy_group='):
            group_str = line[len('custom_proxy_group='):]
            try:
                groups.append(parse_ini_group(group_str))
            except Exception as e:
                print(f"  ⚠️  Skip group: {group_str[:60]} - {e}", file=sys.stderr)
    
    return {
        'template': template_name,
        'description': description,
        'proxy_groups': groups,
        'rule_sets': rules,
    }

# Templates from subconverter base/config (ACL4SSR Local)
# Naming: acl4ssr_local_xxx to match the existing pattern
templates = {
    'acl4ssr_local': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR.ini',
        'desc': 'ACL4SSR 本地 默认版 分组比较全',
    },
    'acl4ssr_local_mini': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_Mini.ini',
        'desc': 'ACL4SSR 本地 精简版',
    },
    'acl4ssr_local_mini_noauto': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_Mini_NoAuto.ini',
        'desc': 'ACL4SSR 本地 精简版+无自动测速',
    },
    'acl4ssr_local_mini_fallback': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_Mini_Fallback.ini',
        'desc': 'ACL4SSR 本地 精简版+fallback',
    },
    'acl4ssr_local_backcn': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_BackCN.ini',
        'desc': 'ACL4SSR 本地 回国',
    },
    'acl4ssr_local_noapple': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_NoApple.ini',
        'desc': 'ACL4SSR 本地 无苹果分流',
    },
    'acl4ssr_local_noauto': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_NoAuto.ini',
        'desc': 'ACL4SSR 本地 无自动测速',
    },
    'acl4ssr_local_noauto_noapple': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_NoAuto_NoApple.ini',
        'desc': 'ACL4SSR 本地 无自动测速&无苹果分流',
    },
    'acl4ssr_local_nomicrosoft': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_NoMicrosoft.ini',
        'desc': 'ACL4SSR 本地 无微软分流',
    },
    'acl4ssr_local_withgfw': {
        'url': 'https://raw.githubusercontent.com/tindy2013/subconverter/master/base/config/ACL4SSR_WithGFW.ini',
        'desc': 'ACL4SSR 本地 GFW列表',
    },
}

for name, info in templates.items():
    print(f"\nFetching {name}...", file=sys.stderr)
    try:
        req = urllib.request.Request(info['url'], headers={'User-Agent': 'subai'})
        with urllib.request.urlopen(req, timeout=15) as resp:
            content = resp.read().decode('utf-8')
        
        result = convert_ini_to_yaml(content, name, info['desc'])
        
        output_path = f'/root/subai/templates/{name}.yaml'
        with open(output_path, 'w', encoding='utf-8') as f:
            yaml.dump(result, f, allow_unicode=True, default_flow_style=False, sort_keys=False)
        
        gcount = len(result['proxy_groups'])
        rcount = len(result['rule_sets'])
        print(f"  ✅ {name}: {gcount} groups, {rcount} rules → {output_path}", file=sys.stderr)
        
    except Exception as e:
        print(f"  ❌ {name}: {e}", file=sys.stderr)

print("\nDone!", file=sys.stderr)