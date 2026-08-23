package store

import (
	"testing"

	"task186-namemerge/internal/model"
)

func TestTask186Bug04_ReverseRelationPairIsConflict(t *testing.T) {
	db, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rs := NewRelationStore(db)
	first := model.NameRelation{ID: "r1", FromNameID: "name-a", ToNameID: "name-b", Kind: model.RelationKindHomotypic}
	if err := rs.Create(&first); err != nil {
		t.Fatal(err)
	}
	reverse := model.NameRelation{ID: "r2", FromNameID: "name-b", ToNameID: "name-a", Kind: model.RelationKindHomotypic}
	if err := rs.Create(&reverse); err != model.ErrConflict {
		t.Fatalf("reverse relation error = %v, want ErrConflict", err)
	}
}
