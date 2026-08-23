package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"task186-namemerge/internal/service"
	"task186-namemerge/internal/store"
)

func TestTask186Bug09_APIRejectsDirectAcceptedTransition(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := service.NewApp(db)
	name, err := app.RegisterName("Quercus robur L.")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/names/"+name.ID,
		bytes.NewBufferString(`{"status":"accepted"}`))
	res := httptest.NewRecorder()
	NewServer(app).Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("direct accepted transition status = %d, want %d; body=%s", res.Code, http.StatusUnprocessableEntity, res.Body.String())
	}
	updated, err := app.Names.Get(name.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated.Status) != "pending_review" {
		t.Fatalf("status after rejected transition = %q, want pending_review", updated.Status)
	}
}
