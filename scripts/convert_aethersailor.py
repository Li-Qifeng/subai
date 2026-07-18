#!/usr/bin/env python3
"""Convert Aethersailor/Custom_OpenClash_Rules .ini templates to subai YAML format."""
import re, sys, yaml

def parse_ini_rule(rule):
    """Parse a ruleset line: ruleset=group,rule"""
    # Split on first comma
    idx = rule.index(',')
    group = rule[:idx].strip()
    r = rule[idx+1:].strip()
    
    # Parse rule format
    if r.startswith('[]FINAL'):
        return {'group': group, 'rule': 'MATCH,' + group}
    elif r.startswith('[]GEOSITE,'):
        geosite = r[len('[]GEOSITE,'):]
        # Remove ,no-resolve
        geosite = geosite.replace(',no-resolve', '')
        return {'group': group, 'built_in': f'geosite:{geosite}'}
    elif r.startswith('[]GEOIP,'):
        geoip = r[len('[]GEOIP,'):]
        geoip = geoip.replace(',no-resolve', '')
        return {'group': group, 'built_in': f'geoip:{geoip}'}
    elif r.startswith('clash-domain:'):
        url = r[len('clash-domain:'):]
        # Remove interval
        url = re.sub(r',\d+$', '', url)
        return {'group': group, 'url': url}
    elif r.startswith('clash-classic:'):
        url = r[len('clash-classic:'):]
        url = re.sub(r',\d+$', '', url)
        return {'group': group, 'url': url}
    else:
        return {'group': group, 'rule': r}

def parse_ini_group(line):
    """Parse a custom_proxy_group line."""
    # Format: name`type`filter_or_proxies`[url`interval,,tolerance]`
    parts = line.split('`')
    name = parts[0].strip()
    gtype = parts[1].strip()
    
    rest = parts[2] if len(parts) > 2 else ''
    
    group = {'name': name, 'type': gtype}
    
    if gtype == 'select':
        # Parse proxies: []proxyname`[]proxyname2`...
        proxies = []
        for item in rest.split('`'):
            item = item.strip()
            if item.startswith('[]'):
                proxies.append(item[2:])
            elif item == '.*':
                pass  # filter all
        if proxies:
            group['proxies'] = proxies
    elif gtype == 'url-test':
        # Format: filter`url`interval,,tolerance
        filter_part = rest
        url_part = parts[3] if len(parts) > 3 else ''
        interval_part = parts[4] if len(parts) > 4 else ''
        
        # Filter is a regex in parentheses
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
    in_rules = False
    in_groups = False
    
    for line in ini_content.split('\n'):
        line = line.strip()
        if not line or line.startswith(';'):
            continue
        
        # Check section markers
        if line.startswith('[custom]'):
            continue
        
        # Parse rulesets
        if line.startswith('ruleset='):
            rule_str = line[len('ruleset='):]
            try:
                rules.append(parse_ini_rule(rule_str))
            except Exception as e:
                print(f"  ⚠️  Skip rule: {rule_str[:60]} - {e}", file=sys.stderr)
        
        # Parse proxy groups
        if line.startswith('custom_proxy_group='):
            group_str = line[len('custom_proxy_group='):]
            try:
                groups.append(parse_ini_group(group_str))
            except Exception as e:
                print(f"  ⚠️  Skip group: {group_str[:60]} - {e}", file=sys.stderr)
    
    # Build YAML
    result = {
        'template': template_name,
        'description': description,
        'proxy_groups': groups,
        'rule_sets': rules,
    }
    return result

# Process all 3 templates
templates = {
    'aethersailor_full': {
        'url': 'https://raw.githubusercontent.com/Aethersailor/Custom_OpenClash_Rules/main/cfg/Custom_Clash_Full.ini',
        'desc': 'Aethersailor/Custom_OpenClash_Rules 全分组防 DNS 泄漏模板 (30+ 策略组)'
    },
    'aethersailor_lite': {
        'url': 'https://raw.githubusercontent.com/Aethersailor/Custom_OpenClash_Rules/main/cfg/Custom_Clash_Lite.ini',
        'desc': 'Aethersailor/Custom_OpenClash_Rules 轻量版分流模板 (15 策略组)'
    },
    'aethersailor_gfw': {
        'url': 'https://raw.githubusercontent.com/Aethersailor/Custom_OpenClash_Rules/main/cfg/Custom_Clash_GFW.ini',
        'desc': 'Aethersailor/Custom_OpenClash_Rules GFW 模式模板 (仅 3 策略组)'
    },
}

import urllib.request

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