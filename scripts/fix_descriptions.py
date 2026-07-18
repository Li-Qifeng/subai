#!/usr/bin/env python3
"""Convert old-style templates to new format with template: + description: fields."""
import json, os, yaml, re

templates_dir = "/root/subai/templates"

with open(f"{templates_dir}/index.json") as f:
    idx = json.load(f)

fixed = 0
for e in idx:
    path = f"{templates_dir}/{e['file']}"
    if not os.path.exists(path):
        continue

    with open(path) as f:
        content = f.read()

    # Skip if already has template: field (new format)
    if re.search(r'^template:', content, re.MULTILINE):
        continue

    # Parse old format: comments + --- + YAML
    desc = e.get("description", "")
    name = e["name"]

    # Find the --- separator
    if '---' in content:
        # Remove old comment header, keep everything after ---
        yaml_part = content.split('---', 1)[1].strip()
    else:
        yaml_part = content.strip()

    # Rebuild in new format
    new_content = f"template: {name}\ndescription: {desc}\n{yaml_part}\n"

    with open(path, 'w') as f:
        f.write(new_content)

    fixed += 1
    print(f"  ✅ {e['file']}")

print(f"\n✅ Fixed {fixed} templates")