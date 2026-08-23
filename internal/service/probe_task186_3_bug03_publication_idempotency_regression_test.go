package service

import (
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func TestTask186Bug03_PublicationFingerprintIgnoresFormatting(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db)
	year := 1753
	first, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus, C.", Journal: "Sp. Pl.",
		YearRangeStart: &year, YearRangeEnd: &year,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.RegisterPublication(model.Publication{
		Title: "  species   plantarum ", Authors: "linnaeus, c.", Journal: " sp. pl. ",
		YearRangeStart: &year, YearRangeEnd: &year,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("formatting variant created a second publication: first=%s second=%s", first.ID, second.ID)
	}
}
