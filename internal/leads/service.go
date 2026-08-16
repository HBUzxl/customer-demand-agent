// service.go 提供商机面板（Dashboard）HTTP 代理用的结构化查询入口：
// List / Stats 与工具层（tools.go）共用同一客户端、限流与包络容错解析，
// 但返回 Go 类型而非回填字符串——HTTP 边界需要干净的 JSON 与错误分类
// （APIError/TransportError 原样上抛，由 http 层映射给面板用户）。
package leads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// LeadList 是线索列表的统一结构化形状（与 leads_search 的回填 JSON 同构）。
// Total 为 nil 表示平台未返回总数（面板据此隐藏翻页上限）。
type LeadList struct {
	Count int               `json:"count"`
	Total *int              `json:"total"`
	Page  int               `json:"page"`
	Items []json.RawMessage `json:"items"`
}

// List 检索线索列表（GET /api/leads；stage 空则不过滤，page/page_size 归一化
// 且封顶 MaxPageSize——参数纪律与 leads_search 完全一致：仅 stage + 分页）。
// 包络识别不出时，整个原始响应作为 items 单元素透传（面板兜底可见，不吞数据）。
func (s *Service) List(ctx context.Context, stage string, page, pageSize int) (*LeadList, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	q := url.Values{}
	if stage != "" {
		q.Set("stage", stage)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	raw, err := s.client.Get(ctx, "/api/leads?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if lst, ok := parseList(raw, page); ok {
		return lst, nil
	}
	return newList(1, nil, page, []json.RawMessage{raw}), nil
}

// Stats 查询统计（metric 白名单与 leads_stats 一致）。
// 错误原样返回（含 APIError 403 权限语义），由调用方决定映射方式。
func (s *Service) Stats(ctx context.Context, metric string) (json.RawMessage, error) {
	ep, ok := statsEndpoints[metric]
	if !ok {
		return nil, fmt.Errorf("metric 必须是 summary/conversion/trends/ml_creators/product_conversion 之一，got %q", metric)
	}
	raw, err := s.client.Get(ctx, ep)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// parseList 是列表包络容错解析（shapeList 与 List 共用）：
// 顶层数组 / 对象中任一常见 items 字段名 → 统一形状；识别不出返回 ok=false。
func parseList(raw []byte, page int) (*LeadList, bool) {
	// 形状 1：顶层即数组
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return newList(len(arr), nil, page, arr), true
	}
	// 形状 2：对象，找常见 items 字段 + 总数字段
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, k := range itemsKeys {
			if v, ok := obj[k]; ok {
				if err := json.Unmarshal(v, &arr); err == nil {
					var total *int
					for _, tk := range totalKeys {
						if tv, ok := obj[tk]; ok {
							var n int
							if err := json.Unmarshal(tv, &n); err == nil {
								total = &n
							}
							break
						}
					}
					return newList(len(arr), total, page, arr), true
				}
			}
		}
	}
	return nil, false
}

// newList 构造 LeadList（items 恒非 nil，JSON 输出 [] 而非 null）。
func newList(count int, total *int, page int, items []json.RawMessage) *LeadList {
	if items == nil {
		items = []json.RawMessage{}
	}
	return &LeadList{Count: count, Total: total, Page: page, Items: items}
}
