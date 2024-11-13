package variables_test

import (
	"testing"

	"github.com/Ensono/taskctl/pkg/variables"
)

func TestNewVariables(t *testing.T) {
	vars1 := variables.FromMap(map[string]string{"a": "1", "b": "2"})

	if vars1.Get("a") != "1" {
		t.Fatal("get test failed")
	}

	vars2 := variables.NewVariables()
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

	if !vars2.Has("d") {
		t.Fatal()
	}

	// test overwrite
	vars2.Set("a", "overwritten")
	varsMergedOverwrite := vars1.Merge(vars2)
	if varsMergedOverwrite.Get("a") != "overwritten" {
		t.Fatalf("merge test overwrite failed, got %v, want: 'overwritten'", varsMergedOverwrite.Get("a"))
	}

}

func TestVariables_MergeV2(t *testing.T) {
	ttests := map[string]struct {
		currentVar    *variables.Variables
		overwriteVars *variables.Variables
		expect        struct {
			key string
			val string
		}
	}{
		"value is overwritten": {
			currentVar:    variables.FromMap(map[string]string{"original": "ignore", "untouched": "foo"}),
			overwriteVars: variables.FromMap(map[string]string{"original": "new"}),
			expect: struct {
				key string
				val string
			}{key: "original", val: "new"},
		},
		"value is left": {
			currentVar:    variables.FromMap(map[string]string{"original": "ignore", "untouched": "foo"}),
			overwriteVars: variables.FromMap(map[string]string{"foo": "bar"}),
			expect: struct {
				key string
				val string
			}{key: "original", val: "ignore"},
		},
		"value is merged with nothing to overwrite": {
			currentVar:    variables.FromMap(map[string]string{"original": "ignore", "untouched": "foo"}),
			overwriteVars: variables.FromMap(map[string]string{"foo": "bar"}),
			expect: struct {
				key string
				val string
			}{key: "foo", val: "bar"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			tt.currentVar.MergeV2(tt.overwriteVars)
			val, found := tt.currentVar.Map()[tt.expect.key]
			if !found {
				t.Errorf("not found %s\n", tt.expect.key)
			}
			if val != tt.expect.val {
				t.Errorf("incorrect value set %q, wanted %q", val, tt.expect.val)
			}
		})
	}

	// check chaining
	t.Run("check chaining by precedence", func(t *testing.T) {
		mainVar := variables.FromMap(map[string]string{})
		var2 := variables.FromMap(map[string]string{"foo": "bar", "some": "123"})
		var3 := variables.FromMap(map[string]string{"baz": "qux", "some": "456"})
		mainVar.MergeV2(var2).MergeV2(var3)

		for _, ss := range [][]string{{"foo", "bar"}, {"some", "456"}, {"baz", "qux"}} {
			if val, found := mainVar.Map()[ss[0]]; found {
				if val != ss[1] {
					t.Errorf("wrong value %q, wanted %q\n", val, ss[1])
				}
			} else {
				t.Errorf("key (%q) not found in mainVar map\n", ss[0])
			}
		}
		if len(mainVar.Map()) != 3 {
			t.Errorf("wrong number of keys in map %v\n", mainVar)
		}
	})
}
