package store

import (
	"testing"

	"task186-namemerge/internal/model"
)

func intp(v int) *int { return &v }

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open("")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNameCRUD(t *testing.T) {
	db := openTestDB(t)
	ns := NewNameStore(db)
	n := model.NameRecord{
		ID: "n1", ScientificName: "Quercus robur L.", Genus: "Quercus",
		SpecificEpithet: "robur", Authors: "L.", OrthographicKey: "quercus robur",
	}
	if err := ns.Create(&n); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := ns.Get("n1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ScientificName != "Quercus robur L." {
		t.Errorf("name = %q", got.ScientificName)
	}
	// 唯一冲突：同拼写+作者。
	dup := model.NameRecord{
		ID: "n2", ScientificName: "Quercus robur L.", Genus: "Quercus",
		SpecificEpithet: "robur", Authors: "L.", OrthographicKey: "quercus robur",
	}
	if err := ns.Create(&dup); err != model.ErrConflict {
		t.Errorf("dup err = %v, want ErrConflict", err)
	}
	// 状态转移约束：accepted 不可直接降级。
	_ = ns.SetStatus("n1", model.NameStatusAccepted)
	upd := *got
	upd.Status = model.NameStatusPendingReview
	if err := ns.Update(&upd); err != model.ErrIllegalTransition {
		t.Errorf("update err = %v, want illegal transition", err)
	}
}

func TestPublicationFingerprintUnique(t *testing.T) {
	db := openTestDB(t)
	ps := NewPublicationStore(db)
	p := model.Publication{
		ID: "p1", Title: "T", Authors: "A", Journal: "J",
		YearRangeStart: intp(1800), YearRangeEnd: intp(1800), Fingerprint: "fp-1",
	}
	if err := ps.Create(&p); err != nil {
		t.Fatalf("create: %v", err)
	}
	p2 := p
	p2.ID = "p2"
	if err := ps.Create(&p2); err != model.ErrConflict {
		t.Errorf("dup err = %v, want ErrConflict", err)
	}
	got, err := ps.ByFingerprint("fp-1")
	if err != nil || got.ID != "p1" {
		t.Errorf("by fingerprint: %v %v", got, err)
	}
}

func TestSpecimenLink(t *testing.T) {
	db := openTestDB(t)
	ns := NewNameStore(db)
	ps := NewPublicationStore(db)
	ss := NewSpecimenStore(db)

	_ = ns.Create(&model.NameRecord{ID: "n1", ScientificName: "A a L.", Genus: "A", SpecificEpithet: "a", OrthographicKey: "a a"})
	_ = ps.Create(&model.Publication{ID: "p1", Title: "T", Authors: "X", Journal: "J", Fingerprint: "f"})
	_ = ss.Create(&model.Specimen{ID: "s1", Collector: "C", Number: "1", Institution: "K", Fingerprint: "sf"})

	link, err := ss.Link("n1", "p1", "s1")
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if link.NameID != "n1" {
		t.Errorf("link name = %q", link.NameID)
	}
	// 幂等重复绑定返回同一链接。
	link2, err := ss.Link("n1", "p1", "s1")
	if err != nil {
		t.Fatalf("relink: %v", err)
	}
	if link2.ID != link.ID {
		t.Errorf("relink returned different id: %q vs %q", link2.ID, link.ID)
	}
	fps, err := ss.SpecimenByLink("n1")
	if err != nil || len(fps) != 1 || fps[0] != "sf" {
		t.Errorf("specimen by link = %v, %v", fps, err)
	}
}

func TestSpecimenLinkKeepsExistingSpecimenOnEmptyRebind(t *testing.T) {
	db := openTestDB(t)
	ns := NewNameStore(db)
	ps := NewPublicationStore(db)
	ss := NewSpecimenStore(db)

	_ = ns.Create(&model.NameRecord{ID: "n2", ScientificName: "A a L.", Genus: "A", SpecificEpithet: "a", OrthographicKey: "a a"})
	_ = ps.Create(&model.Publication{ID: "p2", Title: "T", Authors: "X", Journal: "J", Fingerprint: "f2"})
	_ = ss.Create(&model.Specimen{ID: "s2", Collector: "C", Number: "1", Institution: "K", Fingerprint: "sf2"})

	// 第一次绑定：名称 + 发表 + 模式。
	if _, err := ss.Link("n2", "p2", "s2"); err != nil {
		t.Fatalf("link: %v", err)
	}
	// 第二次重复绑定（同一名称+发表）但未提供模式标本：不应清空已绑定模式。
	link2, err := ss.Link("n2", "p2", "")
	if err != nil {
		t.Fatalf("relink empty: %v", err)
	}
	if link2.SpecimenID != "s2" {
		t.Errorf("specimen id after empty rebind = %q, want %q (existing should be preserved)", link2.SpecimenID, "s2")
	}
	fps, err := ss.SpecimenByLink("n2")
	if err != nil || len(fps) != 1 || fps[0] != "sf2" {
		t.Errorf("specimen by link after empty rebind = %v, %v, want [sf2]", fps, err)
	}
	links, _ := ss.LinksByName("n2")
	if !hasTypeLink(links, "n2") {
		t.Errorf("has_type lost after empty rebind: links = %v", links)
	}
}

// hasTypeLink 镜像 evidence.HasType 的判定，避免 store 包循环依赖 evidence。
func hasTypeLink(links []model.NameLink, nameID string) bool {
	for _, l := range links {
		if l.NameID == nameID && l.SpecimenID != "" {
			return true
		}
	}
	return false
}

func TestRelationStatusFlow(t *testing.T) {
	db := openTestDB(t)
	rs := NewRelationStore(db)
	r := model.NameRelation{ID: "r1", FromNameID: "a", ToNameID: "b", Kind: model.RelationKindHomotypic}
	if err := rs.Create(&r); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := rs.SetStatus("r1", model.RelationStatusProven); err != nil {
		t.Fatalf("set proven: %v", err)
	}
	got, _ := rs.Get("r1")
	if got.Status != model.RelationStatusProven {
		t.Errorf("status = %q", got.Status)
	}
	// 对向重复创建 → 唯一冲突。
	r2 := model.NameRelation{ID: "r2", FromNameID: "b", ToNameID: "a", Kind: model.RelationKindHomotypic}
	if err := rs.Create(&r2); err != model.ErrConflict {
		t.Errorf("dup pair err = %v, want ErrConflict", err)
	}
}

func TestViewTransition(t *testing.T) {
	db := openTestDB(t)
	vs := NewViewStore(db)
	v := model.View{ID: "v1", Name: "V", RuleVersion: "r"}
	if err := vs.Create(&v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := vs.SetStatus("v1", model.ViewStatusPublishable); err != nil {
		t.Fatalf("draft→publishable: %v", err)
	}
	if err := vs.SetStatus("v1", model.ViewStatusPublished); err != nil {
		t.Fatalf("publishable→published: %v", err)
	}
	if err := vs.SetStatus("v1", model.ViewStatusDraft); err != model.ErrIllegalTransition {
		t.Errorf("published→draft should be illegal, got %v", err)
	}
	if err := vs.SetStatus("v1", model.ViewStatusSuperseded); err != nil {
		t.Fatalf("published→superseded: %v", err)
	}
}

func TestChecklistSaveAndFrozen(t *testing.T) {
	db := openTestDB(t)
	cs := NewChecklistStore(db)
	chk := model.Checklist{ID: "c1", ViewID: "v1", RuleVersion: "r", Fingerprint: "fp", Status: "frozen"}
	items := []model.ChecklistItem{
		{ChecklistID: "c1", NameID: "n1", ScientificName: "A", Role: "accepted"},
		{ChecklistID: "c1", NameID: "n2", ScientificName: "B", Role: "synonym", AcceptedNameID: "n1"},
	}
	if err := cs.Save(&chk, items); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := cs.Get("c1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Fingerprint != "fp" {
		t.Errorf("fp = %q", got.Fingerprint)
	}
	gotItems, err := cs.Items("c1")
	if err != nil || len(gotItems) != 2 {
		t.Fatalf("items = %d, %v", len(gotItems), err)
	}
}
