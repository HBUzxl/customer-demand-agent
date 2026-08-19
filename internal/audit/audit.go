// Package audit 提供结构化安全审计与脱敏规则（设计文档 §6.4/§8.1/§8.2）。
//
// 控制面事件（注册/登录/退出/改密/登录失败）写入中央身份库 security_audit_logs；
// 租户业务事件（M2 起）经 SetTenantSink 注入的闭包写入各租户 history.db.audit_logs。
// 两类审计都绝不保存密码、Token、API Key 或完整客户原文。
package audit

import (
	"strings"
	"time"

	"customer-demand-agent/internal/identity"
)

// Service 安全审计服务。
type Service struct {
	store    *identity.Store
	tenantFn func(tenantID, actorUserID, action, objType, objID string) error
}

// NewService 创建审计服务（控制面事件落 identity 库）。
func NewService(store *identity.Store) *Service {
	return &Service{store: store}
}

// SetTenantSink 注入租户业务审计落库回调（M2：TenantRuntime 构造时绑定租户）。
func (s *Service) SetTenantSink(fn func(tenantID, actorUserID, action, objType, objID string) error) {
	s.tenantFn = fn
}

// Log 写入一条控制面安全审计（ip/ua 可选，空则忽略）。
func (s *Service) Log(tenantID, actorUserID, action, objType, objID, result, ip, ua string) error {
	if s.store == nil {
		return nil
	}
	return s.store.InsertAuditLog(&identity.AuditLog{
		TenantID: tenantID, ActorUserID: actorUserID, Action: action,
		ObjectType: objType, ObjectID: objID, Result: result,
		IP: ip, UserAgent: ua, CreatedAt: time.Now().UTC(),
	})
}

// Control 适配 identity.Service 的审计回调签名（控制面事件，不含 ip/ua）。
func (s *Service) Control(tenantID, actorUserID, action, objType, objID, result string) {
	_ = s.Log(tenantID, actorUserID, action, objType, objID, result, "", "")
}

// Tenant 写入租户业务审计（经注入 sink；未注入时忽略——M1 单租户过渡期）。
func (s *Service) Tenant(tenantID, actorUserID, action, objType, objID string) {
	if s.tenantFn != nil {
		_ = s.tenantFn(tenantID, actorUserID, action, objType, objID)
	}
}

// Redact 兜底脱敏：超长正文截断到 max（按 rune），并把明显的敏感键值替换为占位。
// 审计写入前统一调用；审计点应优先只记录最小字段，Redact 是第二道防线。
func Redact(s string, max int) string {
	if max > 0 {
		r := []rune(s)
		if len(r) > max {
			s = string(r[:max]) + "…"
		}
	}
	// 命中的键值整段替换，避免误截半段。
	for _, key := range []string{"api_key", "apikey", "password", "token", "secret"} {
		lower := strings.ToLower(s)
		idx := strings.Index(lower, key)
		if idx >= 0 {
			// 从键名起点截断到行尾/逗号/JSON 引号。
			end := idx + len(key)
			for end < len(s) && s[end] != ',' && s[end] != '\n' && s[end] != '"' && s[end] != '}' {
				end++
			}
			s = s[:idx] + key + "=<redacted>" + s[end:]
		}
	}
	return s
}
