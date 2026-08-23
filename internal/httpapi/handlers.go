package httpapi

import (
	"net/http"

	"task186-namemerge/internal/model"
)

// createRelation 提议名称关系。
func (s *Server) createRelation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromNameID string           `json:"from_name_id"`
		ToNameID   string           `json:"to_name_id"`
		Kind       model.RelationKind `json:"kind"`
		Basis      string           `json:"basis"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	rel, err := s.app.ProposeRelation(req.FromNameID, req.ToNameID, req.Kind, req.Basis)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

// listRelations 列出全部关系边。
func (s *Server) listRelations(w http.ResponseWriter, r *http.Request) {
	rels, err := s.app.Relations.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rels)
}

// proveRelation 把提议关系升级为已证明（带证据校验与环检测）。
func (s *Server) proveRelation(w http.ResponseWriter, r *http.Request) {
	if err := s.app.ProveRelation(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	rel, err := s.app.Relations.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

// createRule 创建法规规则版本。
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var rv model.RuleVersion
	if !decodeJSON(w, r, &rv) {
		return
	}
	out, err := s.app.CreateRule(rv)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// listRules 列出规则版本。
func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.app.Rules.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// currentRule 返回当前规则版本。
func (s *Server) currentRule(w http.ResponseWriter, r *http.Request) {
	rv, err := s.app.Rules.Current()
	if err != nil {
		writeErr(w, err)
		return
	}
	if rv == nil {
		writeJSON(w, http.StatusOK, map[string]any{"current": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": rv})
}

// createView 创建分类观点。
func (s *Server) createView(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		RuleVersion  string `json:"rule_version"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	v, err := s.app.CreateView(req.Name, req.RuleVersion)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

// listViews 列出观点。
func (s *Server) listViews(w http.ResponseWriter, r *http.Request) {
	views, err := s.app.Views.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

// evaluateView 观点求值。
func (s *Server) evaluateView(w http.ResponseWriter, r *http.Request) {
	ev, err := s.app.EvaluateView(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

// viewClusters 查询观点名称簇。
func (s *Server) viewClusters(w http.ResponseWriter, r *http.Request) {
	ev, err := s.app.EvaluateView(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev.Clusters)
}

// viewConflicts 查询观点冲突。
func (s *Server) viewConflicts(w http.ResponseWriter, r *http.Request) {
	ev, err := s.app.EvaluateView(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev.Conflicts)
}

// createRuling 记录裁决。
func (s *Server) createRuling(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RelationID string            `json:"relation_id"`
		Decision   model.RulingDecision `json:"decision"`
		Rationale  string            `json:"rationale"`
		RuledBy    string            `json:"ruled_by"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ru, err := s.app.Ruling(r.PathValue("id"), req.RelationID, req.Decision, req.Rationale, req.RuledBy)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ru)
}

// publishView 发布观点清单。
func (s *Server) publishView(w http.ResponseWriter, r *http.Request) {
	chk, err := s.app.PublishView(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, chk)
}

// viewChecklist 获取观点已发布清单。
func (s *Server) viewChecklist(w http.ResponseWriter, r *http.Request) {
	chk, items, err := s.app.ViewChecklist(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checklist": chk, "items": items})
}

// listChecklists 列出全部清单。
func (s *Server) listChecklists(w http.ResponseWriter, r *http.Request) {
	chks, err := s.app.Checklists.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chks)
}

// checklistDiff 对比两份清单差异。
func (s *Server) checklistDiff(w http.ResponseWriter, r *http.Request) {
	vs := r.URL.Query().Get("vs")
	if vs == "" {
		writeErr(w, model.ErrInvalidArgument)
		return
	}
	d, err := s.app.CompareChecklists(vs, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// stats 返回统计。
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.GetStats()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// health 自检。
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.app.SelfCheck())
}
