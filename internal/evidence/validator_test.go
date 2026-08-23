package evidence

import (
	"testing"

	"task186-namemerge/internal/model"
)

func intp(v int) *int { return &v }

func TestFingerprintStable(t *testing.T) {
	a := model.Publication{Title: "Species Plantarum", Authors: "Linnaeus", Journal: "Sp. Pl."}
	b := model.Publication{Title: "  Species Plantarum ", Authors: "linnaeus", Journal: "Sp. Pl."}
	if FingerprintPublication(a) != FingerprintPublication(b) {
		t.Error("fingerprint should be case/space insensitive")
	}
}

func TestFingerprintDistinct(t *testing.T) {
	a := model.Publication{Title: "A", Authors: "X", Journal: "J"}
	b := model.Publication{Title: "B", Authors: "X", Journal: "J"}
	if FingerprintPublication(a) == FingerprintPublication(b) {
		t.Error("different titles should differ")
	}
}

func TestSortable(t *testing.T) {
	a := model.Publication{Title: "a", Authors: "x", Journal: "j", YearRangeStart: intp(1753), YearRangeEnd: intp(1753)}
	b := model.Publication{Title: "b", Authors: "x", Journal: "j", YearRangeStart: intp(1790), YearRangeEnd: intp(1790)}
	earlierA, ok := Sortable(a, b)
	if !ok {
		t.Fatal("should be sortable")
	}
	if !earlierA {
		t.Error("a (1753) should be earlier")
	}
}

func TestSortableOverlapUnsortable(t *testing.T) {
	a := model.Publication{Title: "a", Authors: "x", Journal: "j", YearRangeStart: intp(1753), YearRangeEnd: intp(1760)}
	b := model.Publication{Title: "b", Authors: "x", Journal: "j", YearRangeStart: intp(1755), YearRangeEnd: intp(1770)}
	if _, ok := Sortable(a, b); ok {
		t.Error("overlapping ranges should be unsortable")
	}
}

func TestSortableMissingDate(t *testing.T) {
	a := model.Publication{Title: "a", Authors: "x", Journal: "j"}
	b := model.Publication{Title: "b", Authors: "x", Journal: "j", YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	if _, ok := Sortable(a, b); ok {
		t.Error("missing date should be unsortable")
	}
}

func TestValidatePublication(t *testing.T) {
	p := model.Publication{Title: "T", Authors: "A", Journal: "J", YearRangeStart: intp(1800), YearRangeEnd: intp(1800)}
	st, err := ValidatePublication(p, true)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if st != model.PublicationStatusValid {
		t.Errorf("status = %q, want valid", st)
	}
	st2, _ := ValidatePublication(p, false)
	if st2 != model.PublicationStatusMissingType {
		t.Errorf("status = %q, want missing_type", st2)
	}
	p2 := model.Publication{Title: "", Authors: "A", Journal: "J"}
	if _, err := ValidatePublication(p2, true); err == nil {
		t.Error("empty title should fail")
	}
}

func TestHasType(t *testing.T) {
	links := []model.NameLink{
		{NameID: "n1", SpecimenID: ""},
		{NameID: "n2", SpecimenID: "s1"},
	}
	if HasType("n1", links) {
		t.Error("n1 has no specimen")
	}
	if !HasType("n2", links) {
		t.Error("n2 has specimen")
	}
}
