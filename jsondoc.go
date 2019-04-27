package jsondoc

import (
	"encoding/json"
	"fmt"
	"reflect"
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

func (d *Atlas) Describe(thing interface{}) string {
	desc, err := json.MarshalIndent(d.describe(reflect.TypeOf(thing)), "", "  ")
	if err != nil {
		panic(err)
	}
	return string(desc)
}

func baseType(ty reflect.Type) reflect.Type {
	for ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	return ty
}

func (d *Atlas) describe(t reflect.Type) interface{} {
	t = baseType(t)
	switch t.Kind() {
	case reflect.Interface:
		desc, ok := d.descriptions[t]
		if !ok {
			desc = "<object>"
		}
		return desc
	case reflect.Map:
		return map[interface{}]interface{}{d.describe(t.Key()): d.describe(t.Elem())}
	case reflect.Struct:
		if desc, ok := d.descriptions[t]; ok {
			return desc
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
		return desc
	case reflect.Slice:
		return []interface{}{d.describe(t.Elem())}
	default:
		return fmt.Sprintf("<%s>", t.Kind())
	}
}
