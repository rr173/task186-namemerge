package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/service"
	"task186-namemerge/internal/store"
)

func TestTask186Bug10_EvidenceEndpointReportsMissingDateAsUnsortable(t *testing.T) {
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
	pub, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus", Journal: "Sp. Pl.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.BindEvidence(name.ID, pub.ID, ""); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/names/"+name.ID+"/evidence", nil)
	res := httptest.NewRecorder()
	NewServer(app).Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("evidence endpoint status = %d, want 200; body=%s", res.Code, res.Body.String())
	}
	var body struct {
		Sortable bool `json:"sortable"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Sortable {
		t.Fatal("evidence without publication dates reported sortable=true")
	}
}
