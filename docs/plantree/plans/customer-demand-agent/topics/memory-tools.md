# Memory Function Calling 工具

> AI Agent 常驻上下文中的记忆管理工具。命名参考 LLM Wiki 工具风格：`verb_object`，一眼能看出干什么。

## 工具总览

| 工具 | 作用 | 对标 wiki 工具 |
|------|------|---------------|
| `memory_search` | 搜索记忆 | `wiki_search` |
| `memory_ensure` | 创建或更新记忆条目 | `wiki_ensure_page` |
| `memory_observe` | 记录轻量洞察 | `wiki_observe` |
| `memory_delete` | 删除/归档记忆条目 | — |
| `memory_recall` | 语义召回（模糊搜索） | `wiki_recall` |
| `memory_list` | 列出某类型全部条目 | — |

## 一、memory_search

**作用**：确定性关键词搜索，跨记忆类型。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `query` | string | ✅ | 搜索关键词 |
| `type` | string | ❌ | 限定类型。不传搜全部。可选：`product` `threat` `compliance` `industry` `customer` `user` `all` |
| `limit` | int | ❌ | 最大返回数，默认 10 |

**返回**：命中的记忆条目列表，每条含类型、标题、摘要。

**示例**：

```
memory_search(query="CC攻击", type="threat")
→ [{type: "threat", title: "CC攻击", summary: "应用层DDoS..."}]

memory_search(query="雷池")
→ [{type: "product", title: "雷池", ...}, {type: "threat", title: "CC攻击", ...关联产品包含雷池}]
```

---

## 二、memory_ensure

**作用**：创建或更新一条记忆（upsert）。核心写工具。如果标题已存在则更新，不存在则创建。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `type` | string | ✅ | 记忆类型：`product` `threat` `compliance` `industry` `customer` `user` |
| `title` | string | ✅ | 条目标题（唯一标识）。如 "雷池"、"CC攻击"、"客户A" |
| `content` | string | ✅ | Markdown 格式的正文 |
| `tags` | string[] | ❌ | 标签列表。产品用 `["WAF", "边界安全"]`，威胁用 `["DDoS", "应用层"]` |
| `aliases` | string[] | ❌ | 别名。如雷池的 `["WAF", "SafeLine"]` |

**返回**：创建或更新后的条目。

**示例**：

```
memory_ensure(
  type="customer",
  title="某某制造集团",
  content="## 行业\n制造业\n\n## 技术栈\nJava + Spring Boot\n\n## 已有安全能力\n- 阿里云基础WAF\n\n## 已知痛点\n- 官网被扫描频繁\n- 等保二级待过",
  tags=["制造业", "等保二级"]
)
```

**注意**：`type=product` 时，仅产品经理或人工可写。AI 写入产品记忆会被拒绝（返回权限错误）。这是 ADR-005 的约束。

---

## 三、memory_observe

**作用**：记录一条轻量观察/洞察。不创建完整页面，只追加一条带时间戳的注记。类比 `wiki_observe`。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `type` | string | ✅ | 关联的记忆类型 |
| `title` | string | ✅ | 洞察标题（≤80 字） |
| `content` | string | ✅ | 洞察正文 |
| `relevance` | string | ❌ | 重要程度：`low` `medium` `high` `critical`。默认 `medium` |
| `tags` | string[] | ❌ | 标签 |

**返回**：创建的观察记录。

**示例**：

```
memory_observe(
  type="threat",
  title="新型API滥用模式",
  content="客户描述的'接口被大量异常调用'不符合标准CC攻击特征，更像是针对API限流漏洞的利用。暂归入'API滥用'，待确认是否需要新建威胁类型。",
  relevance="high",
  tags=["API安全", "新型威胁"]
)
```

---

## 四、memory_delete

**作用**：删除或归档一条记忆。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `type` | string | ✅ | 记忆类型 |
| `title` | string | ✅ | 条目标题 |
| `archive` | bool | ❌ | 是否归档而非真删。默认 `true`（软删除） |

**返回**：操作确认。

**示例**：

```
memory_delete(type="customer", title="某某制造集团", archive=true)
→ "已归档客户画像：某某制造集团"
```

---

## 五、memory_recall

**作用**：语义搜索。当关键词匹配不到时，用自然语言描述来找到"差不多"的记忆。类比 `wiki_recall`。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `query` | string | ✅ | 自然语言描述 |
| `type` | string | ❌ | 限定类型 |
| `max_results` | int | ❌ | 最大返回数，默认 5 |

**返回**：语义匹配的条目列表。

**示例**：

```
memory_recall(query="有没有类似客户网站被扫但说不出具体症状的", type="customer")
→ [{type: "customer", title: "某某电商平台", relevance: 0.87}, ...]
```

**与 memory_search 的区别**：`search` 靠标签/关键词精确匹配，`recall` 靠语义相似度。`recall` 慢但"找感觉"，`search` 快且准。

---

## 六、memory_list

**作用**：列出某类型的所有记忆条目。

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `type` | string | ✅ | 记忆类型 |
| `offset` | int | ❌ | 分页偏移，默认 0 |
| `limit` | int | ❌ | 每页条数，默认 50 |

**返回**：条目摘要列表。

**示例**：

```
memory_list(type="threat")
→ [{title: "CC攻击", tags: ["DDoS"]}, {title: "SQL注入", tags: ["Web安全"]}, ...]
```

---

## 权限矩阵

| 记忆类型 | AI 可读 | AI 可写 | AI 可删 | 说明 |
|----------|:---:|:---:|:---:|------|
| `product` | ✅ | ❌ | ❌ | 产品能力由人维护 |
| `threat` | ✅ | ✅ (打"待审核"标记) | ❌ | AI 发现新模式可记录 |
| `compliance` | ✅ | ✅ (打"待审核"标记) | ❌ | AI 发现新解读可记录 |
| `industry` | ✅ | ✅ (打"待审核"标记) | ❌ | AI 发现新痛点可记录 |
| `customer` | ✅ | ✅ | ✅ | AI 全权维护客户画像 |
| `user` | ✅ | ❌ | ❌ | 使用者画像由管理员维护 |
