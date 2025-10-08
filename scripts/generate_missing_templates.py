#!/usr/bin/env python3
"""
批量生成缺失的模板
"""

import json
import re
import yaml
from collections import defaultdict


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
                        tags.add((event, tag))
    return tags


def extract_event_types(yaml_path):
    """从 events.yaml 提取所有事件和类型"""
    with open(yaml_path, 'r', encoding='utf-8') as f:
        data = yaml.safe_load(f)

    event_types = defaultdict(list)
    for event, config in data.get('events', {}).items():
        if config is None:
            event_types[event] = ['']
        else:
            types = config.get('types', [])
            if types:
                event_types[event] = types
            else:
                event_types[event] = ['']

    return event_types


# 颜色映射
COLOR_MAP = {
    # PR/Issue 相关
    'opened': 'green',
    'created': 'green',
    'published': 'green',
    'fixed': 'green',
    'resolved': 'green',
    'approved': 'green',
    'completed': 'green',
    'success': 'green',

    'closed': 'red',
    'deleted': 'red',
    'removed': 'red',
    'failure': 'red',

    'edited': 'blue',
    'updated': 'blue',
    'synchronize': 'blue',

    'locked': 'orange',
    'dismissed': 'orange',
    'requested': 'orange',

    'unlocked': 'green',
    'reopened': 'orange',

    # 默认
    '': 'turquoise',
    'default': 'turquoise'
}


def get_color(action):
    """根据动作获取颜色"""
    if action in COLOR_MAP:
        return COLOR_MAP[action]
    # 启发式匹配
    if 'success' in action or 'approved' in action or 'fixed' in action:
        return 'green'
    if 'fail' in action or 'error' in action or 'delete' in action:
        return 'red'
    if 'warning' in action or 'pending' in action:
        return 'yellow'
    if 'cancel' in action:
        return 'orange'
    if 'progress' in action or 'running' in action:
        return 'blue'
    return 'turquoise'


def generate_template(event, tag, is_chinese=False):
    """生成单个模板"""
    color = get_color(tag if tag else '')

    # 事件名映射
    event_names_cn = {
        'branch_protection_configuration': '分支保护配置',
        'branch_protection_rule': '分支保护规则',
        'check_run': '检查运行',
        'check_suite': '检查套件',
        'code_scanning_alert': '代码扫描警报',
        'commit_comment': '提交评论',
        'create': '创建',
        'custom_property': '自定义属性',
        'custom_property_values': '自定义属性值',
        'delete': '删除',
        'dependabot_alert': 'Dependabot 警报',
        'deploy_key': '部署密钥',
        'deployment': '部署',
        'deployment_protection_rule': '部署保护规则',
        'deployment_review': '部署审查',
        'deployment_status': '部署状态',
        'issue_comment': 'Issue 评论',
        'issue_dependencies': 'Issue 依赖',
        'label': '标签',
        'marketplace_purchase': '市场购买',
        'member': '成员',
        'membership': '成员资格',
        'merge_group': '合并组',
        'meta': '元数据',
        'milestone': '里程碑',
        'org_block': '组织封禁',
        'organization': '组织',
        'personal_access_token_request': '个人访问令牌请求',
        'project': '项目',
        'project_card': '项目卡片',
        'project_column': '项目列',
        'projects_v2': '项目 V2',
        'projects_v2_item': '项目 V2 条目',
        'projects_v2_status_update': '项目 V2 状态更新',
        'public': '公开',
        'registry_package': '注册表包',
        'repository': '仓库',
        'repository_advisory': '仓库公告',
        'repository_dispatch': '仓库调度',
        'repository_import': '仓库导入',
        'repository_ruleset': '仓库规则集',
        'repository_vulnerability_alert': '仓库漏洞警报',
        'secret_scanning_alert': '密钥扫描警报',
        'secret_scanning_alert_location': '密钥扫描警报位置',
        'secret_scanning_scan': '密钥扫描',
        'security_advisory': '安全公告',
        'security_and_analysis': '安全与分析',
        'sponsorship': '赞助',
        'star': '星标',
        'status': '状态',
        'sub_issues': '子 Issue',
        'team': '团队',
        'team_add': '团队添加',
        'watch': '关注',
        'workflow_dispatch': '工作流调度',
        'workflow_job': '工作流作业',
    }

    # Action 中文映射
    action_names_cn = {
        'created': '已创建',
        'deleted': '已删除',
        'edited': '已编辑',
        'opened': '已打开',
        'closed': '已关闭',
        'reopened': '已重新打开',
        'locked': '已锁定',
        'unlocked': '已解锁',
        'completed': '已完成',
        'requested': '已请求',
        'approved': '已批准',
        'rejected': '已拒绝',
        'dismissed': '已驳回',
        'fixed': '已修复',
        'resolved': '已解决',
        'published': '已发布',
        'updated': '已更新',
        'enabled': '已启用',
        'disabled': '已禁用',
        'added': '已添加',
        'removed': '已移除',
        'transferred': '已转移',
        'renamed': '已重命名',
        'archived': '已归档',
        'unarchived': '已取消归档',
        'publicized': '已公开',
        'privatized': '已私有化',
        'pinned': '已固定',
        'unpinned': '已取消固定',
        'labeled': '已添加标签',
        'unlabeled': '已移除标签',
        'milestoned': '已添加里程碑',
        'demilestoned': '已移除里程碑',
        'assigned': '已分配',
        'unassigned': '已取消分配',
        'in_progress': '进行中',
        'queued': '已排队',
        'waiting': '等待中',
        'success': '成功',
        'failure': '失败',
        'cancelled': '已取消',
        'appeared_in_branch': '出现在分支',
        'closed_by_user': '被用户关闭',
        'reopened_by_user': '被用户重新打开',
        'auto_dismissed': '自动驳回',
        'auto_reopened': '自动重新打开',
        'reintroduced': '重新引入',
        'publicly_leaked': '公开泄露',
        'validated': '已验证',
        'checks_requested': '已请求检查',
        'destroyed': '已销毁',
        'converted': '已转换',
        'moved': '已移动',
        'reordered': '已重新排序',
        'restored': '已恢复',
        'answered': '已回答',
        'unanswered': '未回答',
        'blocked': '已封禁',
        'unblocked': '已解除封禁',
        'member_added': '成员已添加',
        'member_invited': '成员已邀请',
        'member_removed': '成员已移除',
        'added_to_repository': '已添加到仓库',
        'removed_from_repository': '已从仓库移除',
        'suspend': '已暂停',
        'unsuspend': '已恢复',
        'revoked': '已撤销',
        'new_permissions_accepted': '新权限已接受',
        'submitted': '已提交',
        'typed': '已分类',
        'untyped': '已取消分类',
        'blocked_by_added': '被阻止者已添加',
        'blocked_by_removed': '被阻止者已移除',
        'blocking_added': '阻止者已添加',
        'blocking_removed': '阻止者已移除',
        'parent_issue_added': '父 Issue 已添加',
        'parent_issue_removed': '父 Issue 已移除',
        'sub_issue_added': '子 Issue 已添加',
        'sub_issue_removed': '子 Issue 已移除',
        'requested_action': '已请求操作',
        'rerequested': '已重新请求',
        'started': '已开始',
        'withdrawn': '已撤回',
        'reported': '已报告',
        'pending_cancellation': '待取消',
        'pending_tier_change': '待变更等级',
        'tier_changed': '等级已变更',
        'changed': '已变更',
        'pending_change': '待变更',
        'pending_change_cancelled': '待变更已取消',
        'purchased': '已购买',
        'create': '创建',
        'dismiss': '驳回',
        'reopen': '重新打开',
        'resolve': '解决',
        'promote_to_enterprise': '提升至企业',
    }

    event_name = event_names_cn.get(
        event, event) if is_chinese else event.replace('_', ' ').title()
    action_name = action_names_cn.get(
        tag, tag) if is_chinese else tag.replace('_', ' ').title()

    if is_chinese:
        title = f"{event_name} {action_name}" if tag else f"{event_name}"
        repo_label = "仓库："
        action_label = "操作："
        user_label = "用户："
        button_text = "查看详情"
    else:
        title = f"{event_name} {action_name}" if tag else f"{event_name} Event"
        repo_label = "Repository:"
        action_label = "Action:"
        user_label = "User:"
        button_text = "View Details"

    template = {
        "tags": [tag] if tag else ["default"],
        "payload": {
            "msg_type": "interactive",
            "card": {
                "config": {
                    "wide_screen_mode": True
                },
                "header": {
                    "title": {
                        "tag": "plain_text",
                        "content": f"🔔 {title}"
                    },
                    "template": color
                },
                "elements": [
                    {
                        "tag": "div",
                        "text": {
                            "tag": "lark_md",
                            "content": f"**{repo_label}** {{{{repository_link_md}}}}\n**{action_label}** {{{{action}}}}\n**{user_label}** {{{{sender_link_md}}}}"
                        }
                    },
                    {
                        "tag": "hr"
                    },
                    {
                        "tag": "action",
                        "actions": [
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": button_text
                                },
                                "url": "{{repository.html_url}}",
                                "type": "default"
                            }
                        ]
                    }
                ]
            }
        }
    }

    return template


def main():
    print("=" * 80)
    print("批量生成缺失模板")
    print("=" * 80)

    # Load data
    event_types = extract_event_types('configs/events.yaml')
    en_templates = load_jsonc('configs/templates.jsonc')
    cn_templates = load_jsonc('configs/templates.cn.jsonc')

    # Extract existing tags
    en_tags = extract_template_tags(en_templates)
    cn_tags = extract_template_tags(cn_templates)

    # Calculate expected combinations
    expected_combinations = set()
    for event, types in event_types.items():
        for type_ in types:
            expected_combinations.add((event, type_))

    # Find missing
    missing_en = expected_combinations - en_tags
    missing_cn = expected_combinations - cn_tags

    print(f"\n需要生成的英文模板: {len(missing_en)}")
    print(f"需要生成的中文模板: {len(missing_cn)}")

    # Generate templates
    if missing_en:
        print("\n生成英文模板...")
        # Group by event
        missing_by_event = defaultdict(list)
        for event, tag in sorted(missing_en):
            missing_by_event[event].append(tag)

        generated_en = {}
        for event in sorted(missing_by_event.keys()):
            tags = missing_by_event[event]
            templates = []
            for tag in sorted(tags):
                templates.append(generate_template(
                    event, tag, is_chinese=False))
            generated_en[event] = {"payloads": templates}

        # Save to file
        with open('configs/generated_missing_templates_en.json', 'w', encoding='utf-8') as f:
            json.dump({"templates": generated_en}, f,
                      indent=2, ensure_ascii=False)
        print(
            f"✅ 已生成 {len(missing_en)} 个英文模板到 configs/generated_missing_templates_en.json")

    if missing_cn:
        print("\n生成中文模板...")
        # Group by event
        missing_by_event = defaultdict(list)
        for event, tag in sorted(missing_cn):
            missing_by_event[event].append(tag)

        generated_cn = {}
        for event in sorted(missing_by_event.keys()):
            tags = missing_by_event[event]
            templates = []
            for tag in sorted(tags):
                templates.append(generate_template(
                    event, tag, is_chinese=True))
            generated_cn[event] = {"payloads": templates}

        # Save to file
        with open('configs/generated_missing_templates_cn.json', 'w', encoding='utf-8') as f:
            json.dump({"templates": generated_cn}, f,
                      indent=2, ensure_ascii=False)
        print(
            f"✅ 已生成 {len(missing_cn)} 个中文模板到 configs/generated_missing_templates_cn.json")

    print("\n" + "=" * 80)
    print("✅ 完成！请review生成的模板，然后手动合并到主模板文件中")
    print("=" * 80)


if __name__ == '__main__':
    main()
