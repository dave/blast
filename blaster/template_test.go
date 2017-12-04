package blaster

import (
	"reflect"
	"testing"
	"time"
)

func TestRenderNil(t *testing.T) {
	r, err := parseRenderer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Fatal("r should be nil")
	}
}

func TestRender(t *testing.T) {
	data := map[string]string{
		"foo": "FOO",
		"bar": "BAR",
		"baz": "BAZ",
	}
	tmpl := map[string]interface{}{
		"str": "{{.foo}}",
		"a":   "b",
		"c":   1,
		"arr": []interface{}{
			"{{.bar}}",
			"c",
			2,
			time.Second,
		},
		"map": map[string]interface{}{
			"baz": "{{.baz}}",
			"d":   3,
			"t":   time.Second,
		},
		"t": time.Second,
	}
	r, err := parseRenderer(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.render(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{"t": nil, "str": "FOO", "a": "b", "c": 1, "arr": []interface{}{"BAR", "c", 2, nil}, "map": map[string]interface{}{"d": 3, "t": nil, "baz": "BAZ"}}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("Not expected: %#v.", out)
	}
}
