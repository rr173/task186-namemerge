package httpapi

import (
	"net/http"

	"task186-namemerge/internal/evidence"
	"task186-namemerge/internal/model"
	"task186-namemerge/internal/namerecord"
)

// createName 登记名称。
func (s *Server) createName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScientificName string `json:"scientific_name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	n, err := s.app.RegisterName(req.ScientificName)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

// listNames 列出全部名称。
func (s *Server) listNames(w http.ResponseWriter, r *http.Request) {
	names, err := s.app.Names.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, names)
}

// getName 名称详情。
func (s *Server) getName(w http.ResponseWriter, r *http.Request) {
	n, err := s.app.Names.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// updateName 更新名称字段（含状态转移约束）。
func (s *Server) updateName(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cur, err := s.app.Names.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	var req struct {
		ScientificName *string      `json:"scientific_name"`
		Status         *model.NameStatus `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ScientificName != nil {
		parsed, err := namerecord.Parse(*req.ScientificName)
		if err != nil {
			writeErr(w, err)
			return
		}
		cur.ScientificName = parsed.ScientificName
		cur.Genus = parsed.Genus
		cur.SpecificEpithet = parsed.SpecificEpithet
		cur.Authors = parsed.Authors
		cur.OrthographicKey = parsed.OrthographicKey
	}
	if req.Status != nil {
		cur.Status = *req.Status
	}
	if err := s.app.Names.Update(cur); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cur)
}

// createPublication 登记发表证据。
func (s *Server) createPublication(w http.ResponseWriter, r *http.Request) {
	var p model.Publication
	if !decodeJSON(w, r, &p) {
		return
	}
	out, err := s.app.RegisterPublication(p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// listPublications 列出发表证据。
func (s *Server) listPublications(w http.ResponseWriter, r *http.Request) {
	pubs, err := s.app.Publications.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pubs)
}

// createSpecimen 登记模式标本。
func (s *Server) createSpecimen(w http.ResponseWriter, r *http.Request) {
	var sp model.Specimen
	if !decodeJSON(w, r, &sp) {
		return
	}
	out, err := s.app.RegisterSpecimen(sp)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// listSpecimens 列出模式标本。
func (s *Server) listSpecimens(w http.ResponseWriter, r *http.Request) {
	specs, err := s.app.Specimens.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, specs)
}

// bindEvidence 绑定名称 → 发表 → 模式。
func (s *Server) bindEvidence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicationID string `json:"publication_id"`
		SpecimenID    string `json:"specimen_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	link, err := s.app.BindEvidence(r.PathValue("id"), req.PublicationID, req.SpecimenID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

// nameEvidence 名称证据详情：发表 + 模式指纹 + 状态判定。
func (s *Server) nameEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := s.app.Names.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	pub, err := s.app.Publications.ByName(id)
	if err != nil && err != model.ErrNotFound {
		writeErr(w, err)
		return
	}
	links, err := s.app.Specimens.LinksByName(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	fps, _ := s.app.Specimens.SpecimenByLink(id)
	hasType := evidence.HasType(id, links)
	resp := map[string]any{
		"name":            n,
		"publication":     pub,
		"specimen_fingerprints": fps,
		"has_type":        hasType,
		"sortable":        pub != nil,
	}
	writeJSON(w, http.StatusOK, resp)
}
