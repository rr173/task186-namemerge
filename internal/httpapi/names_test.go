package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/service"
	"task186-namemerge/internal/store"
)

func intp(v int) *int { return &v }

// newTestServer 构造一个使用内存 SQLite 的 HTTP 服务用于集成测试。
func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.Open("")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewServer(service.NewApp(db))
}

// doEvidence 请求 /api/names/{id}/evidence 并解析响应。
func doEvidence(t *testing.T, s *Server, id string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/names/"+id+"/evidence", nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evidence status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode evidence: %v", err)
	}
	return out
}

// TestNameEvidenceNotSortableWhenDateMissing 验证名称证据接口在发表记录缺少
// 日期（date_conflict）时不应报告为可排序。
func TestNameEvidenceNotSortableWhenDateMissing(t *testing.T) {
	s := newTestServer(t)
	app := s.app

	// 登记名称（带发表年份，但发表证据本身不填日期）。
	n, err := app.RegisterName("Fagus sylvatica L.")
	if err != nil {
		t.Fatalf("register name: %v", err)
	}
	// 发表证据：仅标题/作者/期刊，无年份区间 → date_conflict，不可排序。
	pub, err := app.RegisterPublication(model.Publication{
		Title: "Sp. Pl.", Authors: "Linnaeus, C.", Journal: "Salvius",
	})
	if err != nil {
		t.Fatalf("register publication: %v", err)
	}
	if _, err := app.BindEvidence(n.ID, pub.ID, ""); err != nil {
		t.Fatalf("bind evidence: %v", err)
	}

	out := doEvidence(t, s, n.ID)
	if sortable, ok := out["sortable"].(bool); !ok || sortable {
		t.Errorf("sortable = %v, want false when publication date missing", out["sortable"])
	}
}

// TestNameEvidenceSortableWhenDatePresent 对照组：发表记录日期完整时应报告为可排序。
func TestNameEvidenceSortableWhenDatePresent(t *testing.T) {
	s := newTestServer(t)
	app := s.app

	n, err := app.RegisterName("Quercus robur L.")
	if err != nil {
		t.Fatalf("register name: %v", err)
	}
	y := 1753
	pub, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus, C.", Journal: "Sp. Pl.",
		YearRangeStart: &y, YearRangeEnd: &y,
	})
	if err != nil {
		t.Fatalf("register publication: %v", err)
	}
	sp, err := app.RegisterSpecimen(model.Specimen{Collector: "Smith", Number: "1", Institution: "K"})
	if err != nil {
		t.Fatalf("register specimen: %v", err)
	}
	if _, err := app.BindEvidence(n.ID, pub.ID, sp.ID); err != nil {
		t.Fatalf("bind evidence: %v", err)
	}

	out := doEvidence(t, s, n.ID)
	if sortable, ok := out["sortable"].(bool); !ok || !sortable {
		t.Errorf("sortable = %v, want true when publication date present", out["sortable"])
	}
}

// TestNameEvidenceNotSortableWhenNoPublication 无发表证据时不得报告为可排序。
func TestNameEvidenceNotSortableWhenNoPublication(t *testing.T) {
	s := newTestServer(t)
	app := s.app

	n, err := app.RegisterName("Quercus pedunculata Ehrh.")
	if err != nil {
		t.Fatalf("register name: %v", err)
	}

	out := doEvidence(t, s, n.ID)
	if sortable, ok := out["sortable"].(bool); !ok || sortable {
		t.Errorf("sortable = %v, want false when no publication bound", out["sortable"])
	}
}

// 防止未引用导入被裁剪（intp 在发表证据年份构造中复用）。
var _ = intp
