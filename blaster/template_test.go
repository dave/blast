package blaster

import (
	"reflect"
	"testing"
)

func TestRender(t *testing.T) {
	data := map[string]string{
		"foo": "FOO",
		"bar": "BAR",
		"baz": "BAZ",
	}
	tmpl := map[string]interface{}{
		"str": "{{.foo}}",
		"arr": []interface{}{
			"{{.bar}}",
		},
		"map": map[string]interface{}{
			"baz": "{{.baz}}",
		},
	}
	r, err := parseRenderer(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.render(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{"map": map[string]interface{}{"baz": "BAZ"}, "str": "FOO", "arr": []interface{}{"BAR"}}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("Actual %#v not equal to expected %#v.", out, expected)
	}
}
