// Package httpapi 提供 HTTP 路由层：把 service.App 的能力
// 暴露为 /api 前缀的 JSON 接口，统一错误映射与响应封装。
package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/service"
)

// Server 封装 HTTP 服务。
type Server struct {
	app *service.App
	mux *http.ServeMux
}

// NewServer 构造 HTTP 服务并注册全部路由。
func NewServer(app *service.App) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler { return s.mux }

// routes 注册全部路由。
func (s *Server) routes() {
	// 名称
	s.mux.HandleFunc("POST /api/names", s.createName)
	s.mux.HandleFunc("GET /api/names", s.listNames)
	s.mux.HandleFunc("GET /api/names/{id}", s.getName)
	s.mux.HandleFunc("PUT /api/names/{id}", s.updateName)
	// 发表
	s.mux.HandleFunc("POST /api/publications", s.createPublication)
	s.mux.HandleFunc("GET /api/publications", s.listPublications)
	// 模式
	s.mux.HandleFunc("POST /api/specimens", s.createSpecimen)
	s.mux.HandleFunc("GET /api/specimens", s.listSpecimens)
	// 证据绑定
	s.mux.HandleFunc("POST /api/names/{id}/evidence", s.bindEvidence)
	s.mux.HandleFunc("GET /api/names/{id}/evidence", s.nameEvidence)
	// 关系
	s.mux.HandleFunc("POST /api/relations", s.createRelation)
	s.mux.HandleFunc("GET /api/relations", s.listRelations)
	s.mux.HandleFunc("POST /api/relations/{id}/prove", s.proveRelation)
	// 规则
	s.mux.HandleFunc("POST /api/rules", s.createRule)
	s.mux.HandleFunc("GET /api/rules", s.listRules)
	s.mux.HandleFunc("GET /api/rules/current", s.currentRule)
	// 观点
	s.mux.HandleFunc("POST /api/views", s.createView)
	s.mux.HandleFunc("GET /api/views", s.listViews)
	s.mux.HandleFunc("POST /api/views/{id}/evaluate", s.evaluateView)
	s.mux.HandleFunc("GET /api/views/{id}/clusters", s.viewClusters)
	s.mux.HandleFunc("GET /api/views/{id}/conflicts", s.viewConflicts)
	s.mux.HandleFunc("POST /api/views/{id}/rulings", s.createRuling)
	s.mux.HandleFunc("POST /api/views/{id}/publish", s.publishView)
	s.mux.HandleFunc("GET /api/views/{id}/checklist", s.viewChecklist)
	// 清单
	s.mux.HandleFunc("GET /api/checklists", s.listChecklists)
	s.mux.HandleFunc("GET /api/checklists/{id}/diff", s.checklistDiff)
	// 系统
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
}

// writeJSON 统一 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// writeErr 统一错误响应。
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrConflict),
		errors.Is(err, model.ErrCycleSynonym),
		errors.Is(err, model.ErrMutuallyExclusiveType),
		errors.Is(err, model.ErrFrozenChecklist):
		status = http.StatusConflict
	case errors.Is(err, model.ErrInvalidArgument),
		errors.Is(err, model.ErrDateUnsortable):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrIllegalTransition),
		errors.Is(err, model.ErrCrossViewMerge):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON 解析请求体。
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, model.ErrInvalidArgument)
		return false
	}
	return true
}
