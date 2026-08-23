package service

import (
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func TestTask186Bug02_EmptyRebindPreservesTypeEvidence(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db)
	name, err := app.RegisterName("Quercus robur L.")
	if err != nil {
		t.Fatal(err)
	}
	year := 1753
	pub, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus", Journal: "Sp. Pl.",
		YearRangeStart: &year, YearRangeEnd: &year,
	})
	if err != nil {
		t.Fatal(err)
	}
	sp, err := app.RegisterSpecimen(model.Specimen{Collector: "Smith", Number: "1234", Institution: "K"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.BindEvidence(name.ID, pub.ID, sp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.BindEvidence(name.ID, pub.ID, ""); err != nil {
		t.Fatal(err)
	}
	links, err := app.Specimens.LinksByName(name.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].SpecimenID != sp.ID {
		t.Fatalf("rebind changed type evidence: %+v, want specimen %s", links, sp.ID)
	}
}
