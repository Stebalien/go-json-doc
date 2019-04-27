package jsondoc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Atlas struct {
	descriptions map[reflect.Type]interface{}
}

func NewAtlas() *Atlas {
	return &Atlas{
		descriptions: make(map[reflect.Type]interface{}),
	}
}

func (d *Atlas) RegisterStructure(thing interface{}, structure interface{}) *Atlas {
	json, err := json.Marshal(structure)
	if err != nil {
		panic(err)
	}
	d.descriptions[baseType(reflect.TypeOf(thing))] = json
	return d
}

func (d *Atlas) RegisterName(thing interface{}, name string) *Atlas {
	d.descriptions[baseType(reflect.TypeOf(thing))] = fmt.Sprintf("<%s>", name)
	return d
}

type state struct {
	complete bool
	desc     interface{}
}

type describeState struct {
	*Atlas
	state map[reflect.Type]*state
}

func (d *Atlas) Describe(thing interface{}) string {
	state := describeState{d, make(map[reflect.Type]*state)}
	desc := state.describe(reflect.TypeOf(thing))
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(desc)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func baseType(ty reflect.Type) reflect.Type {
	for ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	return ty
}
func (d *describeState) describe(t reflect.Type) interface{} {
	t = baseType(t)
	s, ok := d.state[t]
	if ok {
		if !s.complete {
			return "<recursive>"
		}
		return s.desc
	} else {
		s = new(state)
		d.state[t] = s
	}
	switch t.Kind() {
	case reflect.Interface:
		var ok bool
		s.desc, ok = d.descriptions[t]
		if !ok {
			s.desc = "<object>"
		}
	case reflect.Map:
		s.desc = map[string]interface{}{d.describe(t.Key()).(string): d.describe(t.Elem())}
	case reflect.Struct:
		var ok bool
		s.desc, ok = d.descriptions[t]
		if ok {
			break
		}
		desc := make(map[string]interface{}, t.NumField())
		for j := 0; j < t.NumField(); j++ {
			f := t.Field(j)
			name, ok := f.Tag.Lookup("json")
			if !ok {
				name = f.Name
			}
			desc[name] = d.describe(f.Type)
		}
		s.desc = desc
	case reflect.Slice:
		s.desc = []interface{}{d.describe(t.Elem())}
	default:
		s.desc = fmt.Sprintf("<%s>", t.Kind())
	}
	s.complete = true
	return s.desc
}
