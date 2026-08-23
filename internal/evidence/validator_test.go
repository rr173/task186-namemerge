package evidence

import (
	"sync"
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

// TestFingerprintConcurrentStable 覆盖并发场景：多个 goroutine 同时计算同一发表证据指纹时，
// 结果必须稳定一致，且不应发生数据竞争或 panic。
// 用 -race 运行验证无竞争。
func TestFingerprintConcurrentStable(t *testing.T) {
	pub := model.Publication{Title: "Species  Plantarum ", Authors: " Linnaeus ", Journal: "Sp. Pl."}
	sp := model.Specimen{Collector: " von Humboldt ", Number: " 1234a ", Institution: " K "}

	wantPub := FingerprintPublication(pub)
	wantSpec := FingerprintSpecimen(sp)

	const goroutines = 64
	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if got := FingerprintPublication(pub); got != wantPub {
					t.Errorf("publication fingerprint drift: got %q want %q", got, wantPub)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if got := FingerprintSpecimen(sp); got != wantSpec {
					t.Errorf("specimen fingerprint drift: got %q want %q", got, wantSpec)
					return
				}
			}
		}()
	}
	wg.Wait()
}
