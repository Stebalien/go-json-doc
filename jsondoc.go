package jsondoc

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strings"
	"time"
)

var (
	stringerType    = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	marshalJsonType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	marshalTextType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
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

func newGlossary(size int) *Glossary {
	return &Glossary{types: make(map[reflect.Type]*typeState, size)}
}

var defaultGlossary = newGlossary(3).
	// go base64 encodes binary
	WithName([]byte(nil), "base64-string").
	// common types
	WithName(new(error), "error").
	WithName(new(time.Time), "timestamp").
	WithName(new(time.Duration), "duration-ns").
	WithName(new(net.Addr), "network-address").
	WithName(new(net.IP), "ip-address").
	WithName(new(net.IPMask), "ip-netmask").
	WithName(new(net.HardwareAddr), "mac-address").
	WithName(new(big.Int), "integer-string").
	WithName(new(big.Float), "float-string").
	WithName(new(big.Rat), "fraction-string")

// NewGlossary creates a new glossary. In addition to th
func NewGlossary() *Glossary {
	return defaultGlossary.Clone()
}

// WithSchema describes 'thing's type with the given schema. The 'schema' must
// marshal to JSON.
//
// This can be used to give types that implement custom json marshallers
// accurate descriptions.
func (d *Glossary) WithSchema(thing interface{}, schema interface{}) *Glossary {
	d.types[baseType(reflect.TypeOf(thing))] = &typeState{
		state: stateFullDone,
		full:  schema,
	}
	return d
}

// WithName names the 'thing's type.
//
// For example, one can name all instances of MyStruct as "<my-struct>" with:
//
//   glossary.Name(new(MyStruct), "my-struct")
//
func (d *Glossary) WithName(thing interface{}, name string) *Glossary {
	d.types[baseType(reflect.TypeOf(thing))] = &typeState{
		state: stateFullDone,
		full:  fmt.Sprintf("<%s>", name),
	}
	return d
}

// Clone clones the glossary. The cloned glossary can be safely used
// concurrently with the original glossary.
func (d *Glossary) Clone() *Glossary {
	clone := newGlossary(len(d.types))
	for t, state := range d.types {
		if state.state != stateFullDone {
			panic("invalid glossary state, did you catch a panic?")
		}
		clone.types[t] = state
	}
	return clone
}

// Describe returns a description for the given type.
func (d *Glossary) Describe(thing interface{}) (string, error) {
	desc := d.describe(reflect.TypeOf(thing))
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(desc); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	} else if t.Implements(marshalTextType) {
		// Handle types with custom _text_ marshallers.
		return stringDesc
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
			name := f.Name
			isString := false
			if tag := f.Tag.Get("json"); tag != "" {
				parts := strings.Split(tag, ",")
				switch parts[0] {
				case "":
				case "-":
					// skip this field
					continue
				default:
					name = parts[0]
				}

				for _, opt := range parts[1:] {
					switch opt {
					case "string":
						isString = true
					}
				}
			}
			var fieldDesc interface{}
			if isString {
				fieldDesc = fmt.Sprintf("<string-%s>", f.Type.Kind())
			} else {
				fieldDesc = d.describe(f.Type)
			}
			structDesc[name] = fieldDesc
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
