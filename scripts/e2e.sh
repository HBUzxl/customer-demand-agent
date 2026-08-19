#!/usr/bin/env bash
# 已废弃：e2e.sh 是单租户时代的业务冒烟（de-tenancy 语义），多租户落地后已不适用：
#   - 所有业务接口需登录 + CSRF（未登录一律 401），本脚本无认证引导；
#   - 其中"X-Tenant-ID 接受但忽略 / 跨租户可访问"的断言与 §16 验收（跨租户一律 404）
#     直接矛盾，继续运行会给出误导性结果；
#   - /api/platform/** 现为平台管理员专用（租户用户 403）。
# 由以下脚本替代：
#   bash scripts/e2e-multitenant.sh        # 多租户端到端冒烟（双租户隔离 + 跨租户 404）
#   bash scripts/e2e-legacy-migration.sh   # 存量数据迁 Legacy + -setup 引导冒烟
# 设计文档：docs/multi-tenant-design.md §16 验收标准。
echo "e2e.sh 已废弃，请改用 scripts/e2e-multitenant.sh 与 scripts/e2e-legacy-migration.sh。"
exit 0
