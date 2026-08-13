package http

import (
	"net/http"
	"strconv"

	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/model"
)

// configResponse is the GET /api/config payload.
type configResponse struct {
	Models []model.ModelConfig `json:"models"`
	Router model.RouterConfig  `json:"router"`
}

// handleConfigGet: GET /api/config（不回传 api_key，留空表示「不修改」）
func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	models := s.registry.All()
	for i := range models {
		models[i].APIKey = "" // 永不回传密钥；PUT 时留空表示不改
	}
	writeJSON(w, http.StatusOK, configResponse{
		Models: models,
		Router: s.modelMgr.Router(),
	})
}

// configPutReq is the PUT /api/config body (full replace of models + router).
type configPutReq struct {
	Models []model.ModelConfig `json:"models"`
	Router *model.RouterConfig `json:"router"`
}

// handleConfigPut: PUT /api/config
// 全量替换模型 + 路由，并回写 config.json（API key 与配置集中存文件）。
func (s *Server) handleConfigPut(w http.ResponseWriter, r *http.Request) {
	var req configPutReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	// 先快照现有真实密钥（reset 前的真相源）
	existing := map[string]string{}
	for _, name := range s.registry.Names() {
		if c, err := s.registry.Get(name); err == nil {
			existing[name] = c.APIKey
		}
	}
	// 合并：传入 api_key 为空 → 保留旧 key（「留空=不改」语义）
	merged := make([]model.ModelConfig, len(req.Models))
	for i, m := range req.Models {
		if m.APIKey == "" {
			m.APIKey = existing[m.Name]
		}
		merged[i] = m
	}
	_ = s.resetRegistry()
	for _, m := range merged {
		if err := s.registry.Register(m); err != nil {
			writeError(w, http.StatusBadRequest, "注册模型 %s: %v", m.Name, err)
			return
		}
	}
	if req.Router != nil {
		s.modelMgr.SetRouter(req.Router)
	}
	// 回写配置文件（合并后的含密钥模型 + 路由），保留 server/wiki/history 设置
	if err := s.store.Update(func(cfg *config.Config) {
		cfg.Models = merged
		if req.Router != nil {
			cfg.Router = *req.Router
		}
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "回写配置文件: %v", err)
		return
	}
	// 响应同样不回传密钥
	for i := range merged {
		merged[i].APIKey = ""
	}
	writeJSON(w, http.StatusOK, configResponse{
		Models: merged,
		Router: s.modelMgr.Router(),
	})
	writeJSON(w, http.StatusOK, configResponse{
		Models: s.registry.All(),
		Router: s.modelMgr.Router(),
	})
}

// resetRegistry clears all registered models (for full-replace config).
func (s *Server) resetRegistry() error {
	for _, name := range s.registry.Names() {
		_ = s.registry.Delete(name)
	}
	return nil
}

// handleSessionList: GET /api/sessions?limit=&offset=
func (s *Server) handleSessionList(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.history.ListSessions(tenant, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(items), "items": items})
}

// handleSessionGet: GET /api/sessions/{id}
func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id := r.PathValue("id")
	det, err := s.history.GetSession(tenant, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, det)
}

// handleSessionDelete: DELETE /api/sessions/{id}
func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	tenant := tenantFrom(r)
	id := r.PathValue("id")
	if err := s.history.DeleteSession(tenant, id); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
