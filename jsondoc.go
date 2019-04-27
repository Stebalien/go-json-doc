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
	objectDesc      = "<object>"
	stringDesc      = "<string>"
	recursiveDesc   = "..."
)

type state int

const (
	stateBuildingFull state = iota
	stateBuildingShallow
	stateShallowDone
	stateFullDone
)

type typeState struct {
	state         state
	shallow, full interface{}
}

func baseType(ty reflect.Type) reflect.Type {
	for ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	return ty
}

// Glossary describes the set of types used in the structures to be described.
//
// This type memoizes and is NOT type-safe.
type Glossary struct {
	types map[reflect.Type]*typeState
}

// NewGlossary creates a new glossary. In addition to th
func NewGlossary() *Glossary {
	return (&Glossary{types: make(map[reflect.Type]*typeState)}).
		RegisterName(new(error), "error").
		RegisterName(new(time.Time), "timestamp").
		RegisterName([]byte(nil), "<base64-string>") // go base64 encodes binary
}

func (d *Glossary) RegisterStructure(thing interface{}, structure interface{}) *Glossary {
	d.types[baseType(reflect.TypeOf(thing))] = &typeState{
		state: stateFullDone,
		full:  structure,
	}
	return d
}

func (d *Glossary) RegisterName(thing interface{}, name string) *Glossary {
	d.types[baseType(reflect.TypeOf(thing))] = &typeState{
		state: stateFullDone,
		full:  fmt.Sprintf("<%s>", name),
	}
	return d
}

// Describe describes the given type.
func (d *Glossary) Describe(thing interface{}) string {
	desc := d.describe(reflect.TypeOf(thing))
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

func (d *Glossary) describe(t reflect.Type) interface{} {
	t = baseType(t)

	// Check for in-progress or finished descriptions
	s, ok := d.types[t]
	if ok {
		switch s.state {
		case stateBuildingFull:
			// We've recursed, build a shallow description that
			// replaces all recursion points with "..."
			s.state = stateBuildingShallow
		case stateBuildingShallow:
			// We're already building the shallow description,
			// return a recursion point ("...").
			return recursiveDesc
		case stateShallowDone:
			// We're recursing but we already have a shallow
			// description, use it.
			return s.shallow
		case stateFullDone:
			// We've already described this type, use it.
			return s.full
		}
	} else if t.Implements(marshalJsonType) {
		// Handle types with custom marshallers.
		return objectDesc
	} else {
		s = new(typeState)
		d.types[t] = s
	}

	// Describe the type
	var desc interface{}
	switch t.Kind() {
	case reflect.Interface:
		desc = objectDesc
	case reflect.Map:
		key, ok := d.describe(t.Key()).(string)
		if !ok {
			// at the end of the day, js keys must be strings.
			key = stringDesc
		}
		desc = map[string]interface{}{key: d.describe(t.Elem())}
	case reflect.Struct:
		structDesc := make(map[string]interface{}, t.NumField())
		for j := 0; j < t.NumField(); j++ {
			f := t.Field(j)
			if f.PkgPath != "" {
				continue // private field, see the reflect docs
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
			desc = stringDesc
		} else {
			desc = structDesc
		}
	case reflect.Slice:
		desc = []interface{}{d.describe(t.Elem())}
	default:
		desc = fmt.Sprintf("<%s>", t.Kind())
	}

	// save the description
	switch s.state {
	case stateBuildingFull, stateShallowDone:
		// We've finished the full description.
		s.full = desc
		s.state = stateFullDone
	case stateBuildingShallow:
		// We've finished a shallow description, now we need to finish
		// the full one.
		s.shallow = desc
		s.state = stateShallowDone
	case stateFullDone:
		// We've already finished this one...
		panic("impossible state")
	}
	return desc
}
