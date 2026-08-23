package namerecord

import (
	"testing"
)

func TestParseBasic(t *testing.T) {
	n, err := Parse("Quercus robur L.")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if n.Genus != "Quercus" {
		t.Errorf("genus = %q", n.Genus)
	}
	if n.SpecificEpithet != "robur" {
		t.Errorf("epithet = %q", n.SpecificEpithet)
	}
	if n.Authors != "L." {
		t.Errorf("authors = %q", n.Authors)
	}
	if n.OrthographicKey == "" {
		t.Error("orthographic key empty")
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("   "); err == nil {
		t.Error("empty should fail")
	}
	if _, err := Parse("Quercus"); err == nil {
		t.Error("single word should fail")
	}
}

func TestOrthographicVariant(t *testing.T) {
	a, _ := Parse("Acer silvestre L.")
	b, _ := Parse("Acer sylvestre L.")
	if a.OrthographicKey != b.OrthographicKey {
		t.Errorf("variant keys differ: %q vs %q", a.OrthographicKey, b.OrthographicKey)
	}
	if !IsVariant(a, b) {
		t.Error("expected variant")
	}
}

func TestVariantSelfNotVariant(t *testing.T) {
	a, _ := Parse("Acer campestre L.")
	if IsVariant(a, a) {
		t.Error("self should not be variant")
	}
}

func TestCombinationChange(t *testing.T) {
	basionym, _ := Parse("Pinus sylvestris L.")
	cc, err := BuildCombination(basionym, "Pinus")
	if err != nil {
		t.Fatalf("combination: %v", err)
	}
	if cc.NewEpithet != "sylvestris" {
		t.Errorf("epithet = %q", cc.NewEpithet)
	}
	if cc.CombinedName == "" {
		t.Error("combined name empty")
	}
}

func TestSameEpithet(t *testing.T) {
	a, _ := Parse("Pinus sylvestris L.")
	b, _ := Parse("Pinus sylvestris var. montana")
	if !SameEpithet(a, b) {
		t.Error("same epithet expected")
	}
}
