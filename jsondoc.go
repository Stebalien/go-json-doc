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

// Glossary describes the set of types used in the structures to be described.
type Glossary struct {
	descriptions map[reflect.Type]interface{}
}

// NewGlossary creates a new glossary. In addition to th
func NewGlossary() *Glossary {
	return (&Glossary{descriptions: make(map[reflect.Type]interface{})}).
		RegisterName(new(error), "error").
		RegisterName(new(time.Time), "timestamp")
}

func (d *Glossary) RegisterStructure(thing interface{}, structure interface{}) *Glossary {
	d.descriptions[baseType(reflect.TypeOf(thing))] = structure
	return d
}

func (d *Glossary) RegisterName(thing interface{}, name string) *Glossary {
	d.descriptions[baseType(reflect.TypeOf(thing))] = fmt.Sprintf("<%s>", name)
	return d
}

type state int

const (
	stateBuildingRecursive state = iota
	stateBuildingShallow
	stateShallowDone
	stateDone
)

type typeState struct {
	state              state
	shallow, recursive interface{}
}

type describeState struct {
	*Glossary
	state map[reflect.Type]*typeState
}

func (d *Glossary) Describe(thing interface{}) string {
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

	// Handle predefined descriptions
	if desc, ok := d.descriptions[t]; ok {
		return desc
	}

	// Handle types with custom marshallers.
	if t.Implements(marshalJsonType) {
		return objectDesc
	}

	// Handle recursive types.
	s, ok := d.state[t]
	if ok {
		switch s.state {
		case stateBuildingRecursive:
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
		case stateDone:
			// We've already described this type, use it.
			return s.recursive
		}
	} else {
		s = new(typeState)
		d.state[t] = s
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
	case stateBuildingRecursive, stateShallowDone:
		// We've finished the recursive description.
		s.recursive = desc
		s.state = stateDone
	case stateBuildingShallow:
		// We've finished a shallow description, now we need to finish
		// the recursive one.
		s.shallow = desc
		s.state = stateShallowDone
	case stateDone:
		// We've already finished this one...
		panic("impossible state")
	}
	return desc
}
