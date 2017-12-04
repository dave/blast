package blaster

import (
	"reflect"
	"testing"
	"time"
)

func TestRand(t *testing.T) {
	for i := 0; i < 100; i++ {
		r := randInt(-5, 5)
		if r.(int) < -5 || r.(int) > 5 {
			t.Fatal("Unexpected:", r)
		}
	}

	for i := 0; i < 100; i++ {
		r := randFloat(-5.0, 5.0)
		if r.(float64) < -5.0 || r.(float64) > 5.0 {
			t.Fatal("Unexpected:", r)
		}
	}

	s := randString(10)
	if len(s.(string)) != 10 {
		t.Fatal("Unexpected:", s)
	}
}

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
