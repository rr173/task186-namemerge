package service

import (
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func intp(v int) *int { return &v }

func newApp(t *testing.T) *App {
	t.Helper()
	db, err := store.Open("")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewApp(db)
}

// TestRegisterPublicationIdempentCaseSpace 锁定修复：
// 同一发表证据仅标题/作者/期刊有大小写或空白差异时，
// 必须保持幂等并复用已有记录（返回同一 ID，不新增行）。
func TestRegisterPublicationIdempotentCaseSpace(t *testing.T) {
	app := newApp(t)

	first, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus, C.",
		Journal: "Sp. Pl.", YearRangeStart: intp(1753), YearRangeEnd: intp(1753),
	})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	// 仅大小写与首尾/内部空白不同。
	dup, err := app.RegisterPublication(model.Publication{
		Title: "  species\t  plantarum\n", Authors: "linnaeus,  c.",
		Journal: "SP. PL.", YearRangeStart: intp(1753), YearRangeEnd: intp(1753),
	})
	if err != nil {
		t.Fatalf("dup register: %v", err)
	}
	if dup.ID != first.ID {
		t.Errorf("expected to reuse publication %q, got new id %q", first.ID, dup.ID)
	}

	pubs, err := app.Publications.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pubs) != 1 {
		t.Errorf("expected 1 publication after idempotent re-register, got %d", len(pubs))
	}
}

// TestRegisterSpecimenIdempotentCaseSpace 模式标本同样适用大小写/空白归一幂等。
func TestRegisterSpecimenIdempotentCaseSpace(t *testing.T) {
	app := newApp(t)

	first, err := app.RegisterSpecimen(model.Specimen{
		Collector: "Smith", Number: "1234", Institution: "K",
	})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	dup, err := app.RegisterSpecimen(model.Specimen{
		Collector: "  smith ", Number: "1234", Institution: "k",
	})
	if err != nil {
		t.Fatalf("dup register: %v", err)
	}
	if dup.ID != first.ID {
		t.Errorf("expected to reuse specimen %q, got new id %q", first.ID, dup.ID)
	}
	specs, err := app.Specimens.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(specs) != 1 {
		t.Errorf("expected 1 specimen after idempotent re-register, got %d", len(specs))
	}
}
