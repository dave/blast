package blaster

import (
	"bytes"
	"text/template"

	"math/rand"
	"time"

	"github.com/pkg/errors"
)

var builtins = template.FuncMap{
	"rand_int":    randInt,
	"rand_string": randString,
	"rand_float":  randFloat,
}

func randInt(from int, to int) interface{} {
	return rand.Intn(to-from) + from
}

func randFloat(from float64, to float64) interface{} {
	return (rand.Float64() * (to - from)) + from
}

func randString(length int) interface{} {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func parseRenderer(in interface{}) (renderer, error) {
	if in == nil {
		return nil, nil
	}
	switch in := in.(type) {
	case map[string]interface{}:
		out := mapR{}
		for k, v := range in {
			p, err := parseRenderer(v)
			if err != nil {
				return nil, err
			}
			out[k] = p
		}
		return out, nil
	case []interface{}:
		out := sliceR{}
		for _, v := range in {
			p, err := parseRenderer(v)
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, nil
	case string:
		tmpl, err := template.New("t").Funcs(builtins).Parse(in)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return templateR{tmpl}, nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
		return nativeR{in}, nil
	default:
		return nil, nil
	}
}

type renderer interface {
	render(data map[string]string) (interface{}, error)
}

type mapR map[string]interface{}

func (m mapR) render(data map[string]string) (interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range m {
		if v, ok := v.(renderer); ok {
			r, err := v.render(data)
			if err != nil {
				return nil, err
			}
			out[k] = r
		} else {
			out[k] = v
		}
	}
	return out, nil
}

type sliceR []interface{}

func (s sliceR) render(data map[string]string) (interface{}, error) {
	out := []interface{}{}
	for _, v := range s {
		if v, ok := v.(renderer); ok {
			r, err := v.render(data)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		} else {
			out = append(out, v)
		}
	}
	return out, nil
}

type templateR struct {
	*template.Template
}

func (t templateR) render(data map[string]string) (interface{}, error) {
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf.String(), nil
}

type native interface{}

type nativeR struct {
	native
}

func (n nativeR) render(data map[string]string) (interface{}, error) {
	return n.native, nil
}
