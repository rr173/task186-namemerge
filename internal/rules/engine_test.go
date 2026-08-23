package rules

import (
	"testing"

	"task186-namemerge/internal/model"
)

func intp(v int) *int { return &v }

func TestIsHomonym(t *testing.T) {
	a := model.NameRecord{ID: "a", ScientificName: "Quercus robur L.", Authors: "L.", OrthographicKey: "quercus robur"}
	b := model.NameRecord{ID: "b", ScientificName: "Quercus robur J.Sm.", Authors: "J.Sm.", OrthographicKey: "quercus robur"}
	pubA := model.Publication{YearRangeStart: intp(1753), YearRangeEnd: intp(1753)}
	pubB := model.Publication{YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	if !IsHomonym(a, b, pubA, pubB) {
		t.Error("b is later homonym of a")
	}
	if IsHomonym(b, a, pubB, pubA) {
		t.Error("a is earlier, not homonym")
	}
}

func TestIsHomonymSameAuthor(t *testing.T) {
	a := model.NameRecord{ID: "a", OrthographicKey: "quercus robur", Authors: "L."}
	b := model.NameRecord{ID: "b", OrthographicKey: "quercus robur", Authors: "L."}
	pubA := model.Publication{YearRangeStart: intp(1753), YearRangeEnd: intp(1753)}
	pubB := model.Publication{YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	if IsHomonym(a, b, pubA, pubB) {
		t.Error("same author is not homonym")
	}
}

func TestEvaluateMissingType(t *testing.T) {
	n := model.NameRecord{ID: "n1", ScientificName: "X y L."}
	pub := model.Publication{Title: "T", Authors: "A", Journal: "J", YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	j := Evaluate(n, pub, false, false)
	if j.Status != model.NameStatusPendingReview {
		t.Errorf("status = %q, want pending_review (missing type)", j.Status)
	}
}

func TestEvaluateValidLegitimate(t *testing.T) {
	n := model.NameRecord{ID: "n1", ScientificName: "X y L."}
	pub := model.Publication{Title: "T", Authors: "A", Journal: "J", YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	j := Evaluate(n, pub, true, false)
	if j.Status != model.NameStatusLegitimate {
		t.Errorf("status = %q, want legitimate", j.Status)
	}
	if !j.DateSortable || !j.PriorityFirst {
		t.Error("date should be sortable and priority first by default")
	}
}

func TestEvaluateDateConflict(t *testing.T) {
	n := model.NameRecord{ID: "n1", ScientificName: "X y L."}
	pub := model.Publication{Title: "T", Authors: "A", Journal: "J"}
	j := Evaluate(n, pub, true, false)
	if !j.Conflicting || j.Status != model.NameStatusPendingReview {
		t.Errorf("date conflict not detected: %+v", j)
	}
}

func TestApplyLegitimacy(t *testing.T) {
	j := Judgment{Status: model.NameStatusLegitimate}
	ApplyLegitimacy(&j, true)
	if j.Status != model.NameStatusIllegitimate {
		t.Errorf("status = %q, want illegitimate", j.Status)
	}
}
