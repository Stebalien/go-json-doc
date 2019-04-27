package jsondoc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	stringerType    = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	marshalJsonType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
)

type Atlas struct {
	descriptions map[reflect.Type]interface{}
}

func NewAtlas() *Atlas {
	return (&Atlas{descriptions: make(map[reflect.Type]interface{})}).
		RegisterName(new(error), "error").
		RegisterName(new(time.Time), "timestamp")
}

func (d *Atlas) RegisterStructure(thing interface{}, structure interface{}) *Atlas {
	d.descriptions[baseType(reflect.TypeOf(thing))] = structure
	return d
}

func (d *Atlas) RegisterName(thing interface{}, name string) *Atlas {
	d.descriptions[baseType(reflect.TypeOf(thing))] = fmt.Sprintf("<%s>", name)
	return d
}

type state int

const (
	stateBuildingComplete state = iota
	stateBuildingShallow
	stateDone
)

type typeState struct {
	state state
	desc  interface{}
}

type describeState struct {
	*Atlas
	state map[reflect.Type]*typeState
}

func (d *Atlas) Describe(thing interface{}) string {
	state := describeState{d, make(map[reflect.Type]*typeState)}
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
		switch s.state {
		case stateBuildingComplete:
			s.state = stateBuildingShallow
		case stateBuildingShallow:
			return "..."
		case stateDone:
			return s.desc
		}
	} else {
		s = new(typeState)
		d.state[t] = s
	}

	var desc interface{}
	switch t.Kind() {
	case reflect.Interface:
		var ok bool
		desc, ok = d.descriptions[t]
		if !ok {
			s.desc = "<object>"
		}
	case reflect.Map:
		s.desc = map[string]interface{}{d.describe(t.Key()).(string): d.describe(t.Elem())}
	case reflect.Struct:
		var ok bool
		desc, ok = d.descriptions[t]
		if ok {
			break
		}

		if t.Implements(marshalJsonType) {
			desc = "<object>"
			break
		}

		structDesc := make(map[string]interface{}, t.NumField())
		for j := 0; j < t.NumField(); j++ {
			f := t.Field(j)
			if f.PkgPath != "" {
				continue // private field
			}
			name := f.Tag.Get("json")
			if idx := strings.IndexByte(name, ','); idx >= 0 {
				name = name[:idx]
			}
			if name == "" {
				name = f.Name
			}
			structDesc[name] = d.describe(f.Type)
		}
		if len(structDesc) == 0 && t.Implements(stringerType) {
			desc = "<string>"
		} else {
			desc = structDesc
		}
	case reflect.Slice:
		desc = []interface{}{d.describe(t.Elem())}
	default:
		desc = fmt.Sprintf("<%s>", t.Kind())
	}
	switch s.state {
	case stateBuildingComplete:
		s.desc = desc
	case stateBuildingShallow:
	case stateDone:
		panic("impossible state")
	}
	return desc
}
