package cluster

import (
	"testing"

	"task186-namemerge/internal/model"
)

func intp(v int) *int { return &v }

func nameRec(id, name string, year int) model.NameRecord {
	y := year
	return model.NameRecord{
		ID: id, ScientificName: name, Status: model.NameStatusPendingReview,
		YearRangeStart: &y, YearRangeEnd: &y,
	}
}

func TestBuildSimpleCluster(t *testing.T) {
	a := nameRec("a", "Quercus robur L.", 1753)
	b := nameRec("b", "Quercus pedunculata Ehrh.", 1790)
	names := []model.NameRecord{a, b}
	pubs := map[string]model.Publication{
		"a": {YearRangeStart: intp(1753), YearRangeEnd: intp(1753)},
		"b": {YearRangeStart: intp(1790), YearRangeEnd: intp(1790)},
	}
	rels := []model.NameRelation{
		{FromNameID: "a", ToNameID: "b", Kind: model.RelationKindHomotypic, Status: model.RelationStatusProven},
	}
	builder := NewBuilder(names, pubs, rels)
	clusters, _, err := builder.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1", len(clusters))
	}
	if clusters[0].AcceptedID != "a" {
		t.Errorf("accepted = %q, want a (earlier)", clusters[0].AcceptedID)
	}
}

func TestBuildCycleRejected(t *testing.T) {
	names := []model.NameRecord{
		nameRec("a", "A", 1800), nameRec("b", "B", 1801), nameRec("c", "C", 1802),
	}
	pubs := map[string]model.Publication{
		"a": {YearRangeStart: intp(1800), YearRangeEnd: intp(1800)},
		"b": {YearRangeStart: intp(1801), YearRangeEnd: intp(1801)},
		"c": {YearRangeStart: intp(1802), YearRangeEnd: intp(1802)},
	}
	rels := []model.NameRelation{
		{FromNameID: "a", ToNameID: "b", Status: model.RelationStatusProven},
		{FromNameID: "b", ToNameID: "c", Status: model.RelationStatusProven},
		{FromNameID: "c", ToNameID: "a", Status: model.RelationStatusProven}, // 成环
	}
	builder := NewBuilder(names, pubs, rels)
	if _, _, err := builder.Build(); err != model.ErrCycleSynonym {
		t.Errorf("err = %v, want ErrCycleSynonym", err)
	}
}

func TestDetectCycle(t *testing.T) {
	acyclic := []model.NameRelation{
		{FromNameID: "a", ToNameID: "b", Status: model.RelationStatusProven},
		{FromNameID: "b", ToNameID: "c", Status: model.RelationStatusProven},
	}
	if DetectCycle(acyclic) {
		t.Error("acyclic detected as cycle")
	}
	cyclic := append(acyclic, model.NameRelation{FromNameID: "c", ToNameID: "a", Status: model.RelationStatusProven})
	if !DetectCycle(cyclic) {
		t.Error("cycle not detected")
	}
}

func TestSharedSpecimenConflicts(t *testing.T) {
	m := map[string][]string{
		"fp1": {"n1", "n2"}, // 一个模式指向两个候选 → 冲突
		"fp2": {"n3"},
	}
	conflicts := SharedSpecimenConflicts(m)
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(conflicts))
	}
}

func TestResolveClusterAccepted(t *testing.T) {
	c := Cluster{Members: []string{"a", "b"}, AcceptedID: "a"}
	accepted, _ := ResolveClusterAccepted(c, model.RulingAccept, "b")
	if accepted != "b" {
		t.Errorf("accepted = %q, want b", accepted)
	}
	accepted2, _ := ResolveClusterAccepted(c, model.RulingDefer, "")
	if accepted2 != "" {
		t.Errorf("defer accepted = %q, want empty", accepted2)
	}
}
