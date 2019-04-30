package jsondoc

import (
	"encoding/json"
	"strings"
	"testing"
)

func test(t *testing.T, gl *Glossary, thing interface{}, expect interface{}) {
	actual, err := gl.Describe(thing)
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(expect); err != nil {
		t.Fatal(err)
	}
	if actual != buf.String() {
		t.Fatalf("bad doc: expected: %s\ngot: %s", buf.String(), actual)
	}
}

func TestBasic(t *testing.T) {
	gl := NewGlossary()
	test(t, gl,
		new(struct {
			A int
			B string
			C string `json:"-"`
			D string `json:"delta"`
			E []byte
			F uint64 `json:",string"`
		}),
		Object{
			"A":     "<int>",
			"B":     "<string>",
			"delta": "<string>",
			"E":     "<base64-string>",
			"F":     "<uint64-string>",
		},
	)
}

type Recursive struct {
	Value    []byte
	Children []Recursive
}

func TestRecursive(t *testing.T) {
	gl := NewGlossary()
	test(t, gl, new(Recursive), Object{
		"Value": "<base64-string>",
		"Children": Array{
			Object{
				"Value":    "<base64-string>",
				"Children": Array{"..."},
			},
		},
	})
}

type MyStruct1 struct{}
type MyStruct2 struct{}

func TestCustom(t *testing.T) {
	gl := NewGlossary().
		WithName(new(MyStruct1), "my-struct").
		WithSchema(new(MyStruct2), Object{"PhantomField": "<my-type>"})

	test(t, gl, new(struct {
		A MyStruct1
		B MyStruct2
	}), Object{
		"A": "<my-struct>",
		"B": Object{
			"PhantomField": "<my-type>",
		},
	})
}
