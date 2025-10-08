#!/usr/bin/env python3
"""对比中英文模板差异"""

import json
import re


def remove_comments(content):
    content = re.sub(r'//.*', '', content)
    content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
    return content


def extract_tags(data):
    if 'templates' in data:
        data = data['templates']
    tags = {}
    for event, event_data in data.items():
        tags[event] = set()
        if 'payloads' in event_data:
            for payload in event_data['payloads']:
                if 'tags' in payload:
                    for tag in payload['tags']:
                        tags[event].add(tag)
    return tags


with open('configs/templates.jsonc', 'r') as f:
    en_data = json.loads(remove_comments(f.read()))

with open('configs/templates.cn.jsonc', 'r') as f:
    cn_data = json.loads(remove_comments(f.read()))

en_tags = extract_tags(en_data)
cn_tags = extract_tags(cn_data)

print('=' * 80)
print('中文模板缺少的内容（相对于英文模板）')
print('=' * 80)
print()

missing_events = []
partial_events = []

for event in sorted(en_tags.keys()):
    if event not in cn_tags:
        missing_events.append((event, sorted(en_tags[event])))
    else:
        missing = en_tags[event] - cn_tags[event]
        if missing:
            partial_events.append((event, sorted(missing)))

if missing_events:
    print(f'❌ 完全缺失的事件 ({len(missing_events)} 个):\n')
    for event, tags in missing_events:
        print(f'  - {event}')
        print(f'    Tags: {", ".join(tags)}')
        print()

if partial_events:
    print(f'⚠️  部分缺失的事件 ({len(partial_events)} 个):\n')
    for event, tags in partial_events:
        print(f'  - {event}')
        print(f'    缺少的 tags: {", ".join(tags)}')
        print()

print('=' * 80)
print(f'总计: 完全缺失 {len(missing_events)} 个事件, 部分缺失 {len(partial_events)} 个事件')
print('=' * 80)
