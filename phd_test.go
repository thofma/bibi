package main

import (
	"testing"
	"io/ioutil"
	"github.com/nickng/bibtex"
)

func AssertEntriesEqual(t *testing.T, a, b *bibtex.BibEntry) {
	if a.Type != b.Type {
		t.Error("type mismatch")
	}
	if a.CiteName != b.CiteName {
		t.Error("cite name mismatch")
	}
	if len(a.Fields) != len(b.Fields) {
		t.Fatal("different number of fields")
	}
	for key := range a.Fields {
		if a.Fields[key].String() != b.Fields[key].String() {
			t.Fatalf("mismatch on field %q:\n%v\n%v", key, a.Fields[key].String(), b.Fields[key].String())
		}
	}
}

func TestSingleHit(t *testing.T) {
	data, _ := ioutil.ReadFile("testdata/single_hit.html")
	s := string(data)
	res, _ := MGPResponse(s)

	if len(res) != 1 {
		t.Errorf("Expected to find one hit")
	}
	ent := res[0]
	if ent.author != "Claus Fieker" {
		t.Errorf("Got: %v", ent.author)
	}

	if ent.uni != "Technische Universität Berlin" {
		t.Errorf("Got: %v", ent.uni)
	}

	if ent.year != "1997" {
		t.Errorf("Got: %v", ent.year)
	}

	if ent.title != "Über relative Normgleichungen in algebraischen Zahlkörpern" {
		t.Errorf("Got: %v", ent.title)
	}

	// Test the BibEntry
	expected := bibtex.NewBibTex()
	entry := bibtex.NewBibEntry("thesis", "ClausFieker1997")
	entry.AddField("author", bibtex.NewBibConst("Claus Fieker"))
	entry.AddField("title", bibtex.NewBibConst("{Ü}ber relative {N}ormgleichungen in algebraischen {Z}ahlkörpern"))
	entry.AddField("year", bibtex.NewBibConst("1997"))
	entry.AddField("school", bibtex.NewBibConst("Technische Universität Berlin"))
	expected.AddEntry(entry)

	AssertEntriesEqual(t, entry, ent.bib)
}

func TestMultipleHit(t *testing.T) {
	data, _ := ioutil.ReadFile("testdata/multiple_hits.html")
	s := string(data)
	res, _ := MGPResponse(s)
	if len(res) != 2 {
		t.Errorf("Expected to find one hit")
	}

	ent := res[0]
	if ent.author != "Hofmann, Tobias" {
		t.Errorf("Got: %v", ent.author)
	}
	if ent.uni != "Universität Kassel" {
		t.Errorf("Got: %v", ent.uni)
	}
	if ent.year != "2011" {
		t.Errorf("Got: %v", ent.year)
	}
	if ent.id != "245793" {
		t.Errorf("Got: %v", ent.id)
	}

	ent = res[1]
	if ent.author != "Hofmann, Tobias" {
		t.Errorf("Got: %v", ent.author)
	}
	if ent.uni != "Technische Universität Chemnitz" {
		t.Errorf("Got: %v", ent.uni)
	}
	if ent.year != "2023" {
		t.Errorf("Got: %v", ent.year)
	}
	if ent.id != "302960" {
		t.Errorf("Got: %v", ent.id)
	}

}

