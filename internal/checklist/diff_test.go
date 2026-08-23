package checklist

import (
	"testing"

	"task186-namemerge/internal/model"
)

func TestCompareRoleChanges(t *testing.T) {
	from := model.Checklist{ViewID: "v1", Fingerprint: "fp1"}
	to := model.Checklist{ViewID: "v2", Fingerprint: "fp2"}
	fromItems := []model.ChecklistItem{
		{NameID: "a", ScientificName: "A", Role: "accepted"},
		{NameID: "b", ScientificName: "B", Role: "synonym", AcceptedNameID: "a"},
	}
	toItems := []model.ChecklistItem{
		{NameID: "a", ScientificName: "A", Role: "synonym", AcceptedNameID: "b"},
		{NameID: "b", ScientificName: "B", Role: "accepted"},
	}
	d := Compare(from, fromItems, to, toItems)
	if d.FingerprintSame {
		t.Error("fingerprints differ, expected different")
	}
	if len(d.RoleChanges) != 2 {
		t.Errorf("role changes = %d, want 2", len(d.RoleChanges))
	}
	if len(d.AcceptedAdded) != 1 || d.AcceptedAdded[0] != "b" {
		t.Errorf("accepted added = %v, want [b]", d.AcceptedAdded)
	}
	if len(d.AcceptedLost) != 1 || d.AcceptedLost[0] != "a" {
		t.Errorf("accepted lost = %v, want [a]", d.AcceptedLost)
	}
}

func TestCompareAddRemove(t *testing.T) {
	from := model.Checklist{ViewID: "v1", Fingerprint: "f1"}
	to := model.Checklist{ViewID: "v2", Fingerprint: "f2"}
	fromItems := []model.ChecklistItem{{NameID: "a", ScientificName: "A", Role: "accepted"}}
	toItems := []model.ChecklistItem{
		{NameID: "a", ScientificName: "A", Role: "accepted"},
		{NameID: "c", ScientificName: "C", Role: "deferred"},
	}
	d := Compare(from, fromItems, to, toItems)
	if len(d.Added) != 1 || d.Added[0] != "c" {
		t.Errorf("added = %v, want [c]", d.Added)
	}
	if len(d.Removed) != 0 {
		t.Errorf("removed = %v, want none", d.Removed)
	}
}

func TestCompareIdentical(t *testing.T) {
	from := model.Checklist{ViewID: "v1", Fingerprint: "same"}
	to := model.Checklist{ViewID: "v1", Fingerprint: "same"}
	items := []model.ChecklistItem{
		{NameID: "a", ScientificName: "A", Role: "accepted"},
		{NameID: "b", ScientificName: "B", Role: "synonym"},
	}
	d := Compare(from, items, to, items)
	if !d.FingerprintSame {
		t.Error("same fingerprint expected")
	}
	if len(d.RoleChanges) != 0 || len(d.Added) != 0 || len(d.Removed) != 0 {
		t.Errorf("identical should have no changes: %+v", d)
	}
}
