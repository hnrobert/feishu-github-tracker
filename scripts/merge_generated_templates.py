#!/usr/bin/env python3
"""
自动合并生成的模板到主模板文件
"""

import json
import re
from collections import OrderedDict


def remove_comments(content):
    """移除 JSONC 注释"""
    content = re.sub(r'//.*', '', content)
    content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
    return content


def load_jsonc(filepath):
    """加载 JSONC 文件"""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
        content = remove_comments(content)
        return json.loads(content)


def save_jsonc(filepath, data, is_chinese=False):
    """保存为 JSONC 格式"""
    json_str = json.dumps(data, indent=2, ensure_ascii=False)

    # 添加注释头
    if is_chinese:
        header = """// 中文模板配置文件
// 此文件定义了所有 GitHub 事件的飞书消息卡片模板（中文版）

"""
    else:
        header = """// Template configuration file
// This file defines all Feishu message card templates for GitHub events (English version)

"""

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(header)
        f.write(json_str)


def merge_templates(main_file, generated_file, output_file, is_chinese=False):
    """合并模板"""
    print(f"\n合并 {generated_file} 到 {output_file}...")

    # Load files
    main_data = load_jsonc(main_file)
    generated_data = load_jsonc(generated_file)

    # Extract templates
    if 'templates' in main_data:
        main_templates = main_data['templates']
    else:
        main_templates = main_data

    if 'templates' in generated_data:
        generated_templates = generated_data['templates']
    else:
        generated_templates = generated_data

    # Merge
    merged = OrderedDict()
    all_events = sorted(set(list(main_templates.keys()) +
                        list(generated_templates.keys())))

    stats = {'main': 0, 'generated': 0, 'total': 0}

    for event in all_events:
        if event in main_templates and event in generated_templates:
            # Both exist, merge payloads
            main_payloads = main_templates[event].get('payloads', [])
            generated_payloads = generated_templates[event].get('payloads', [])

            # Get existing tags
            existing_tags = set()
            for payload in main_payloads:
                for tag in payload.get('tags', []):
                    existing_tags.add(tag)

            # Add only new payloads
            merged_payloads = list(main_payloads)
            for payload in generated_payloads:
                tags = payload.get('tags', [])
                if not any(tag in existing_tags for tag in tags):
                    merged_payloads.append(payload)
                    stats['generated'] += 1
                else:
                    stats['main'] += 1

            merged[event] = {'payloads': merged_payloads}
        elif event in main_templates:
            # Only in main
            merged[event] = main_templates[event]
            stats['main'] += len(main_templates[event].get('payloads', []))
        else:
            # Only in generated
            merged[event] = generated_templates[event]
            stats['generated'] += len(
                generated_templates[event].get('payloads', []))

    stats['total'] = stats['main'] + stats['generated']

    # Save
    output_data = {'templates': merged}
    save_jsonc(output_file, output_data, is_chinese)

    print(f"✅ 合并完成:")
    print(f"   - 保留主模板: {stats['main']} 个")
    print(f"   - 新增模板: {stats['generated']} 个")
    print(f"   - 总计: {stats['total']} 个")

    return stats


def main():
    print("=" * 80)
    print("自动合并生成的模板")
    print("=" * 80)

    # Merge English templates
    en_stats = merge_templates(
        'configs/templates.jsonc',
        'configs/generated_missing_templates_en.json',
        'configs/templates.jsonc',
        is_chinese=False
    )

    # Merge Chinese templates
    cn_stats = merge_templates(
        'configs/templates.cn.jsonc',
        'configs/generated_missing_templates_cn.json',
        'configs/templates.cn.jsonc',
        is_chinese=True
    )

    print("\n" + "=" * 80)
    print("📊 合并统计")
    print("=" * 80)
    print(f"\n英文模板:")
    print(f"  - 原有: {en_stats['main']} 个")
    print(f"  - 新增: {en_stats['generated']} 个")
    print(f"  - 总计: {en_stats['total']} 个")

    print(f"\n中文模板:")
    print(f"  - 原有: {cn_stats['main']} 个")
    print(f"  - 新增: {cn_stats['generated']} 个")
    print(f"  - 总计: {cn_stats['total']} 个")

    print("\n" + "=" * 80)
    print("✅ 所有模板已合并！正在验证编译...")
    print("=" * 80)


if __name__ == '__main__':
    main()
