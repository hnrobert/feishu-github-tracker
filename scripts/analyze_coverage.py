#!/usr/bin/env python3
"""
分析模板覆盖率 - 检查哪些事件类型在模板中缺失
"""

import json
import re
from collections import defaultdict

import yaml


def remove_comments(content):
    """移除 JSONC 注释"""
    # Remove single-line comments
    content = re.sub(r'//.*', '', content)
    # Remove multi-line comments
    content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
    return content


def load_jsonc(filepath):
    """加载 JSONC 文件"""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
        content = remove_comments(content)
        return json.loads(content)


def extract_template_tags(data):
    """从模板数据中提取所有 event:tag 组合"""
    if 'templates' in data:
        data = data['templates']
    tags = set()
    for event, event_data in data.items():
        if 'payloads' in event_data:
            for payload in event_data['payloads']:
                if 'tags' in payload:
                    for tag in payload['tags']:
                        # 将 'default' 标签转换为空字符串，以匹配 events.yaml
                        normalized_tag = '' if tag == 'default' else tag
                        tags.add((event, normalized_tag))
    return tags


def extract_event_types(yaml_path):
    """从 events.yaml 提取所有事件和类型"""
    with open(yaml_path, 'r', encoding='utf-8') as f:
        data = yaml.safe_load(f)

    event_types = defaultdict(list)
    for event, config in data.get('events', {}).items():
        if config is None:
            # 无配置的事件，使用空字符串表示默认
            event_types[event] = ['']
        else:
            types = config.get('types', [])
            if types:
                event_types[event] = types
            else:
                # 无 types 的事件，使用空字符串表示默认
                event_types[event] = ['']

    return event_types


def main():
    print("=" * 80)
    print("模板覆盖率分析")
    print("=" * 80)

    # Load data
    print("\n📂 加载配置文件...")
    event_types = extract_event_types('configs/events.yaml')
    en_templates = load_jsonc('configs/templates.jsonc')
    cn_templates = load_jsonc('configs/templates.cn.jsonc')

    # Extract tags
    en_tags = extract_template_tags(en_templates)
    cn_tags = extract_template_tags(cn_templates)

    print(f"✅ Events.yaml: {len(event_types)} events")
    print(f"✅ English templates: {len(en_tags)} event:tag combinations")
    print(f"✅ Chinese templates: {len(cn_tags)} event:tag combinations")

    # Calculate expected combinations
    expected_combinations = set()
    for event, types in event_types.items():
        for type_ in types:
            expected_combinations.add((event, type_))

    print(f"✅ Expected combinations: {len(expected_combinations)}")

    # Find missing
    print("\n" + "=" * 80)
    print("🔍 检查缺失的模板")
    print("=" * 80)

    missing_en = expected_combinations - en_tags
    missing_cn = expected_combinations - cn_tags

    print(f"\n❌ English templates missing: {len(missing_en)}")
    if missing_en:
        # Group by event
        missing_by_event = defaultdict(list)
        for event, tag in sorted(missing_en):
            missing_by_event[event].append(tag)

        for event in sorted(missing_by_event.keys()):
            tags = missing_by_event[event]
            print(
                f"  - {event}: {', '.join(sorted(tags)) if tags[0] else '(default)'}")

    print(f"\n❌ Chinese templates missing: {len(missing_cn)}")
    if missing_cn:
        # Group by event
        missing_by_event = defaultdict(list)
        for event, tag in sorted(missing_cn):
            missing_by_event[event].append(tag)

        for event in sorted(missing_by_event.keys()):
            tags = missing_by_event[event]
            print(
                f"  - {event}: {', '.join(sorted(tags)) if tags[0] else '(default)'}")

    # Check for templates not in events.yaml
    print("\n" + "=" * 80)
    print("⚠️  检查多余的模板（不在 events.yaml 中）")
    print("=" * 80)

    extra_en = en_tags - expected_combinations
    extra_cn = cn_tags - expected_combinations

    if extra_en:
        print(f"\n⚠️  English extra templates: {len(extra_en)}")
        for event, tag in sorted(extra_en):
            print(f"  - {event}:{tag}")
    else:
        print("\n✅ No extra English templates")

    if extra_cn:
        print(f"\n⚠️  Chinese extra templates: {len(extra_cn)}")
        for event, tag in sorted(extra_cn):
            print(f"  - {event}:{tag}")
    else:
        print("\n✅ No extra Chinese templates")

    # Summary
    print("\n" + "=" * 80)
    print("📊 总结")
    print("=" * 80)

    en_coverage = (len(en_tags) / len(expected_combinations)
                   * 100) if expected_combinations else 0
    cn_coverage = (len(cn_tags) / len(expected_combinations)
                   * 100) if expected_combinations else 0

    print(
        f"\n英文模板覆盖率: {en_coverage:.1f}% ({len(en_tags)}/{len(expected_combinations)})")
    print(
        f"中文模板覆盖率: {cn_coverage:.1f}% ({len(cn_tags)}/{len(expected_combinations)})")

    if missing_en or missing_cn:
        print("\n❌ 仍有缺失的模板需要补全！")
        return 1
    else:
        print("\n✅ 所有模板已完整覆盖！")
        return 0


if __name__ == '__main__':
    import sys
    sys.exit(main())
