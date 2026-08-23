package view

import (
	"testing"

	"task186-namemerge/internal/cluster"
	"task186-namemerge/internal/model"
)

func intp(v int) *int { return &v }

func testNames() []model.NameRecord {
	return []model.NameRecord{
		{ID: "a", ScientificName: "Quercus robur L.", Status: model.NameStatusLegitimate},
		{ID: "b", ScientificName: "Quercus pedunculata Ehrh.", Status: model.NameStatusPendingReview},
		{ID: "c", ScientificName: "Fagus sylvatica L.", Status: model.NameStatusPendingReview},
	}
}

func TestEvaluateRoles(t *testing.T) {
	ev := &Evaluator{
		Names: testNames(),
		Publications: map[string]model.Publication{
			"a": {YearRangeStart: intp(1753), YearRangeEnd: intp(1753)},
			"b": {YearRangeStart: intp(1790), YearRangeEnd: intp(1790)},
			"c": {YearRangeStart: intp(1753), YearRangeEnd: intp(1753)},
		},
		Relations: []model.NameRelation{
			{FromNameID: "a", ToNameID: "b", Kind: model.RelationKindHomotypic, Status: model.RelationStatusProven},
		},
		SpecimenToNames:   map[string][]string{},
		HasType:           map[string]bool{"a": true, "b": true, "c": false},
		Rules:             model.RuleVersion{PriorityRule: true, LegitimacyReq: true},
	}
	res, err := ev.Evaluate("v1")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if res.Roles["a"] != "accepted" {
		t.Errorf("a role = %q, want accepted", res.Roles["a"])
	}
	if res.Roles["b"] != "synonym" {
		t.Errorf("b role = %q, want synonym", res.Roles["b"])
	}
	if res.Roles["c"] != "deferred" {
		t.Errorf("c role = %q, want deferred (missing type)", res.Roles["c"])
	}
	// 缺模式产生 missing_type 冲突。
	if len(res.Conflicts) != 1 {
		t.Errorf("conflicts = %v, want 1 missing_type", res.Conflicts)
	}
	if res.Conflicts[0].Kind != "missing_type" {
		t.Errorf("conflict kind = %q, want missing_type", res.Conflicts[0].Kind)
	}
}

func TestEvaluateSpecimenConflict(t *testing.T) {
	ev := &Evaluator{
		Names: testNames(),
		Publications: map[string]model.Publication{
			"a": {YearRangeStart: intp(1753), YearRangeEnd: intp(1753)},
			"b": {YearRangeStart: intp(1790), YearRangeEnd: intp(1790)},
			"c": {YearRangeStart: intp(1753), YearRangeEnd: intp(1753)},
		},
		Relations: []model.NameRelation{},
		SpecimenToNames: map[string][]string{
			"fpK": {"a", "b"}, // 同一模式指向两个名称 → 冲突
		},
		HasType: map[string]bool{"a": true, "b": true, "c": true},
		Rules:   model.RuleVersion{PriorityRule: true},
	}
	res, err := ev.Evaluate("v1")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(res.Conflicts) != 1 {
		t.Errorf("conflicts = %d, want 1", len(res.Conflicts))
	}
	if res.Conflicts[0].Kind != "specimen_conflict" {
		t.Errorf("kind = %q", res.Conflicts[0].Kind)
	}
}

func TestChecklistSnapshotFingerprintStable(t *testing.T) {
	names := testNames()
	nameByID := map[string]model.NameRecord{}
	for _, n := range names {
		nameByID[n.ID] = n
	}
	ev := &Evaluation{
		ViewID: "v1",
		Clusters: []cluster.Cluster{
			{Members: []string{"a", "b"}, AcceptedID: "a", ProvenCount: 1},
		},
		Roles: map[string]string{"a": "accepted", "b": "synonym", "c": "deferred"},
	}
	chk1, items1 := ChecklistSnapshot("v1", "r1", ev, nameByID)
	chk2, _ := ChecklistSnapshot("v1", "r1", ev, nameByID)
	if chk1.Fingerprint != chk2.Fingerprint {
		t.Error("fingerprint not stable for identical input")
	}
	if len(items1) != 3 {
		t.Errorf("items = %d, want 3", len(items1))
	}
}
