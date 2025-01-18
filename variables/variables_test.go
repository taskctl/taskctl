package variables

import (
	"testing"
)

func TestNewVariables(t *testing.T) {
	vars1 := FromMap(map[string]string{"a": "1", "b": "2"})

	if vars1.Get("a") != "1" {
		t.Fatal("get test failed")
	}

	vars2 := NewVariables()
	vars2.Set("c", "3")
	vars2.Set("d", "4")

	vars3 := vars2.With("e", "5")
	if vars3.Get("e") != "5" {
		t.Fatal("with test failed")
	}

	if vars2.Get("d") != "4" || vars3.Get("d") != "4" {
		t.Fatal("with test failed")
	}

	vars1 = vars1.Merge(vars2)
	if vars1.Get("a") != "1" || vars1.Get("c") != "3" {
		t.Fatal("merge test failed")
	}

	if vars2.Get("c") != "3" {
		t.Fatal("merge test failed")
	}

	if vars2.Has("d") == false {
		t.Fatal()
	}
}
