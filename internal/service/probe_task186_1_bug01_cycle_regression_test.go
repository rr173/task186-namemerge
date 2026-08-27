package service

import (
	"errors"
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func TestTask186Bug01_CycleRelationIsRejected(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db)

	years := []int{1753, 1760, 1770}
	names := make([]*model.NameRecord, 3)
	pubs := make([]*model.Publication, 3)
	for i, raw := range []string{"Quercus robur L.", "Quercus robur A.Brown", "Quercus robur C.Doe"} {
		n, err := app.RegisterName(raw)
		if err != nil {
			t.Fatal(err)
		}
		names[i] = n
		y := years[i]
		p, err := app.RegisterPublication(model.Publication{
			Title: "Publication " + raw, Authors: "Author " + raw, Journal: "Journal",
			YearRangeStart: &y, YearRangeEnd: &y,
		})
		if err != nil {
			t.Fatal(err)
		}
		pubs[i] = p
	}
	sp, err := app.RegisterSpecimen(model.Specimen{Collector: "Smith", Number: "42", Institution: "K"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range names {
		if _, err := app.BindEvidence(names[i].ID, pubs[i].ID, sp.ID); err != nil {
			t.Fatal(err)
		}
	}

	pairs := [][2]int{{0, 1}, {1, 2}, {2, 0}}
	for i, pair := range pairs {
		rel, err := app.ProposeRelation(names[pair[0]].ID, names[pair[1]].ID, model.RelationKindHomotypic, "shared specimen")
		if err != nil {
			t.Fatal(err)
		}
		err = app.ProveRelation(rel.ID)
		if i < len(pairs)-1 && err != nil {
			t.Fatalf("proof %d: %v", i, err)
		}
		if i == len(pairs)-1 && !errors.Is(err, model.ErrCycleSynonym) {
			t.Fatalf("closing relation error = %v, want ErrCycleSynonym", err)
		}
	}
}
