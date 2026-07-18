#!/usr/bin/env python3
"""Convert SleepyHeeead/subconverter-config customized .ini templates to subai YAML."""
import re, sys, yaml, urllib.request

def parse_ini_rule(rule):
    idx = rule.index(',')
    group = rule[:idx].strip()
    r = rule[idx+1:].strip()
    if r.startswith('[]FINAL'):
        return {'group': group, 'rule': 'MATCH,' + group}
    elif r.startswith('[]GEOSITE,'):
        return {'group': group, 'built_in': f'geosite:{r[len("[]GEOSITE,"):].replace(",no-resolve","")}'}
    elif r.startswith('[]GEOIP,'):
        return {'group': group, 'built_in': f'geoip:{r[len("[]GEOIP,"):].replace(",no-resolve","")}'}
    elif r.startswith('clash-domain:'):
        return {'group': group, 'url': re.sub(r',\d+$', '', r[len('clash-domain:'):])}
    elif r.startswith('clash-classic:'):
        return {'group': group, 'url': re.sub(r',\d+$', '', r[len('clash-classic:'):])}
    else:
        return {'group': group, 'rule': r}

def parse_ini_group(line):
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

def convert_ini(ini_content, template_name, description):
    rules, groups = [], []
    for line in ini_content.split('\n'):
        line = line.strip()
        if not line or line.startswith(';') or line.startswith('[custom]'):
            continue
        if line.startswith('ruleset='):
            try:
                rules.append(parse_ini_rule(line[len('ruleset='):]))
            except Exception as e:
                print(f"  ⚠️  Skip rule: {line[:60]} - {e}", file=sys.stderr)
        if line.startswith('custom_proxy_group='):
            try:
                groups.append(parse_ini_group(line[len('custom_proxy_group='):]))
            except Exception as e:
                print(f"  ⚠️  Skip group: {line[:60]} - {e}", file=sys.stderr)
    return {'template': template_name, 'description': description,
            'proxy_groups': groups, 'rule_sets': rules}

BASE = 'https://raw.githubusercontent.com/SleepyHeeead/subconverter-config/master/remote-config/customized'

templates = {
    'customized_maying':     {'file': 'maying.ini',     'desc': '定制模板 - Maying'},
    'customized_ytoo':       {'file': 'ytoo.ini',       'desc': '定制模板 - Ytoo'},
    'customized_flowercloud':{'file': 'flowercloud.ini','desc': '定制模板 - FlowerCloud'},
    'customized_nexitally':  {'file': 'nexitally.ini',  'desc': '定制模板 - Nexitally'},
    'customized_socloud':    {'file': 'socloud.ini',    'desc': '定制模板 - SoCloud'},
    'customized_ark':        {'file': 'ark.ini',        'desc': '定制模板 - ARK'},
    'customized_ssrcloud':   {'file': 'ssrcloud.ini',  'desc': '定制模板 - ssrCloud'},
    'customized_nyancat':    {'file': 'nyancat.ini',   'desc': '定制模板 - Nyancat'},
    'customized_xianyu':     {'file': 'xianyu.ini',    'desc': '定制模板 - 咸鱼'},
}

for name, info in templates.items():
    url = f"{BASE}/{info['file']}"
    print(f"\nFetching {name}...", file=sys.stderr)
    try:
        req = urllib.request.Request(url, headers={'User-Agent': 'subai'})
        with urllib.request.urlopen(req, timeout=15) as resp:
            content = resp.read().decode('utf-8')
        result = convert_ini(content, name, info['desc'])
        path = f'/root/subai/templates/{name}.yaml'
        with open(path, 'w', encoding='utf-8') as f:
            yaml.dump(result, f, allow_unicode=True, default_flow_style=False, sort_keys=False)
        g = len(result['proxy_groups'])
        r = len(result['rule_sets'])
        print(f"  ✅ {name}: {g} groups, {r} rules -> {path}", file=sys.stderr)
    except Exception as e:
        print(f"  ❌ {name}: {e}", file=sys.stderr)

print("\nDone!", file=sys.stderr)