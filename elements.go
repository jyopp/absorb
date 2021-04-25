package absorb

import (
	"reflect"
	"strings"
	"sync"
)

type elementBuilder struct {
	Type reflect.Type
	// Keys contains the array of keys, used to get key names for map[string] types.
	Keys []string
	// Field indexes are a *set* of integer indices used to reach a struct field.
	Fields []reflect.StructField
}

var cachedAbsorbers sync.Map

func getBuildersForType(t reflect.Type) *sync.Map {
	i, ok := cachedAbsorbers.Load(t)
	if !ok {
		i, _ = cachedAbsorbers.LoadOrStore(t, &sync.Map{})
	}
	return i.(*sync.Map)
}

func getBuilder(elemTyp reflect.Type, tag string, keys []string) *elementBuilder {
	absorbers := getBuildersForType(elemTyp)

	compoundKey := tag + ":" + strings.Join(keys, "+")
	i, ok := absorbers.Load(compoundKey)
	if !ok {
		toPut := newBuilder(elemTyp, tag, keys)
		i, _ = absorbers.LoadOrStore(compoundKey, toPut)
	}
	return i.(*elementBuilder)
}

func newBuilder(elemTyp reflect.Type, tag string, keys []string) *elementBuilder {
	a := &elementBuilder{
		Type: elemTyp,
		Keys: keys,
	}

	if elemTyp.Kind() == reflect.Struct {
		tagFields := make(map[string]reflect.StructField)
		if tag != "" {
			for i := 0; i < elemTyp.NumField(); i++ {
				field := elemTyp.Field(i)
				if tagVal := field.Tag.Get(tag); tagVal != "" {
					tagFields[tagVal] = field
				}
			}
		}

		fields := make([]reflect.StructField, len(keys))
		for idx, key := range keys {
			if tagged, ok := tagFields[key]; ok {
				fields[idx] = tagged
			} else if field, ok := a.Type.FieldByName(key); ok {
				fields[idx] = field
			}
		}
		a.Fields = fields
	}

	return a
}

// element returns a new element of a's Elem type, constructed from the given values.
func (a *elementBuilder) element(values []interface{}) reflect.Value {
	// Allocate a value for assignment
	dstVal := reflect.Indirect(reflect.New(a.Type))

	switch a.Type.Kind() {
	case reflect.Map:
		// Use the field names directly
		dstVal = reflect.MakeMapWithSize(a.Type, len(values))
		for srcIdx := range values {
			key := reflect.ValueOf(a.Keys[srcIdx])
			val := reflect.ValueOf(values[srcIdx])
			dstVal.SetMapIndex(key, val)
		}
	case reflect.Struct:
		for idx, field := range a.Fields {
			val := reflect.ValueOf(values[idx])
			dstVal.FieldByIndex(field.Index).Set(val)
		}
	default:
		// Expect this to crash often, until we better understand the desired behavior.
		dstVal.Set(reflect.ValueOf(values[0]))
	}

	return dstVal

	// container := reflect.ValueOf(dst)
	// for container.Type().Elem() != dstVal.Type() {
	// 	container = container.Elem()
	// }
}
