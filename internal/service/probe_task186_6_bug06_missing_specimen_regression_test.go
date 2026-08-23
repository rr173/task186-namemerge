package service

import (
	"errors"
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func TestTask186Bug06_MissingSpecimenReturnsDomainError(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db)
	a, err := app.RegisterName("Quercus robur L.")
	if err != nil {
		t.Fatal(err)
	}
	b, err := app.RegisterName("Quercus petraea L.")
	if err != nil {
		t.Fatal(err)
	}
	rel, err := app.ProposeRelation(a.ID, b.ID, model.RelationKindHomotypic, "missing specimen")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ProveRelation(rel.ID); !errors.Is(err, model.ErrInvalidArgument) {
		t.Fatalf("missing specimen error = %v, want ErrInvalidArgument", err)
	}
}
