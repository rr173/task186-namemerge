package service

import (
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/store"
)

func TestTask186Bug05_OrthographicVariantConflicts(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db)
	if _, err := app.RegisterName("Silvestris alba L."); err != nil {
		t.Fatal(err)
	}
	if _, err := app.RegisterName("Sylvestris alba L."); err != model.ErrConflict {
		t.Fatalf("orthographic variant error = %v, want ErrConflict", err)
	}
}
