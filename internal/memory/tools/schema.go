package tools

import "customer-demand-agent/internal/domain"

// 各工具的参数 JSON Schema 定义（OpenAI function-calling 格式）。

func searchDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_search",
		Description: "确定性关键词搜索，跨记忆类型。查产品/威胁/合规/行业/客户知识用这个，快且准。只返回摘要定位；确定目标后用 memory_get 读完整内容。",
		Parameters: obj(
			prop("query", str("搜索关键词"), true),
			prop("type", enumStr("限定类型：product/threat/compliance/industry/customer/user，不传搜全部", false,
				"product", "threat", "compliance", "industry", "customer", "user", "all"), false),
			prop("limit", intProp("最大返回数，默认 10", false), false),
		),
	}
}

func getDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name: "memory_get",
		Description: "按 type+title 获取一条记忆的完整内容：产品主页含全部能力点（带置信度）、" +
			"适用场景、能力边界、竞品对比、相关文档目录与正文；其他类型含结构化字段与正文。" +
			"确定候选产品后、正式推荐或引用细节前必读。正文超长时首次返回章节目录，" +
			"用 section 参数精读指定章节（或 body_offset 续读）。",
		Parameters: obj(
			prop("type", enumStr("记忆类型", true,
				"product", "threat", "compliance", "industry", "customer", "user"), true),
			prop("title", str("条目标题（来自 memory_search 结果或产品目录）"), true),
			prop("section", str("章节名：正文超长时按章节精读（见首次返回的章节目录）"), false),
			prop("body_offset", intProp("正文续读偏移（字符）：无章节结构时的分段兜底", false), false),
		),
	}
}

func ensureDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_ensure",
		Description: "创建或更新一条记忆（upsert）。分析中发现新客户/新威胁/新行业特征时调用。产品记忆(product)不可写。",
		Parameters: obj(
			prop("type", enumStr("记忆类型", true, "threat", "compliance", "industry", "customer"), true),
			prop("title", str("条目标题（唯一标识），如 'CC攻击'、'某某集团'"), true),
			prop("content", str("Markdown 格式正文"), true),
			prop("tags", arrStr("标签列表"), false),
			prop("aliases", arrStr("别名列表"), false),
		),
	}
}

func observeDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_observe",
		Description: "记录一条轻量观察/洞察（不创建完整页面，只追加带时间戳的注记）。发现新型威胁、客户特殊模式时用。",
		Parameters: obj(
			prop("type", enumStr("关联的记忆类型", true, "threat", "compliance", "industry", "customer"), true),
			prop("title", str("洞察标题（≤80 字）"), true),
			prop("content", str("洞察正文"), true),
			prop("relevance", enumStr("重要程度，默认 medium", false, "low", "medium", "high", "critical"), false),
			prop("tags", arrStr("标签"), false),
		),
	}
}

func deleteDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_delete",
		Description: "删除或归档一条记忆。仅客户画像(customer)允许 AI 删除，其余类型返回权限错误。",
		Parameters: obj(
			prop("type", enumStr("记忆类型", true, "customer"), true),
			prop("title", str("条目标题"), true),
			prop("archive", boolProp("是否归档而非真删，默认 true（软删除）"), false),
		),
	}
}

func recallDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_recall",
		Description: "语义召回（模糊搜索）。关键词匹配不到时，用自然语言描述找'差不多'的记忆。比 search 慢但能'找感觉'。",
		Parameters: obj(
			prop("query", str("自然语言描述"), true),
			prop("type", enumStr("限定类型，不传搜全部", false, "product", "threat", "compliance", "industry", "customer", "user"), false),
			prop("max_results", intProp("最大返回数，默认 5", false), false),
		),
	}
}

func listDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name:        "memory_list",
		Description: "列出某类型的全部记忆条目（分页）。想看有哪些产品/威胁/客户时用。",
		Parameters: obj(
			prop("type", enumStr("记忆类型", true, "product", "threat", "compliance", "industry", "customer", "user"), true),
			prop("offset", intProp("分页偏移，默认 0", false), false),
			prop("limit", intProp("每页条数，默认 50", false), false),
		),
	}
}

// ── JSON Schema 构造辅助 ──────────────────────────────────────

func obj(props ...map[string]any) map[string]any {
	required := []string{}
	allProps := map[string]any{}
	for _, p := range props {
		if name, ok := p["__name"].(string); ok {
			req, _ := p["__required"].(bool)
			if req {
				required = append(required, name)
			}
			delete(p, "__name")
			delete(p, "__required")
			allProps[name] = p
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": allProps,
		"required":   required,
	}
}

func prop(name string, schema map[string]any, required bool) map[string]any {
	s := map[string]any{}
	for k, v := range schema {
		s[k] = v
	}
	s["__name"] = name
	s["__required"] = required
	return s
}

func str(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string, _ bool) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func arrStr(desc string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": "string"}}
}

func enumStr(desc string, _ bool, values ...string) map[string]any {
	ev := make([]any, len(values))
	for i, v := range values {
		ev[i] = v
	}
	return map[string]any{"type": "string", "description": desc, "enum": ev}
}
