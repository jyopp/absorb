package absorb

import (
	"reflect"
	"strings"
	"sync"
)

type absorber struct {
	Elem reflect.Type
	// Keys contains the array of keys, used to get key names for map[string] types.
	Keys []string
	// Field indexes are a *set* of integer indices used to reach a struct field.
	Fields []reflect.StructField
}

var cachedAbsorbers sync.Map

func getAbsorbersForType(t reflect.Type) *sync.Map {
	i, ok := cachedAbsorbers.Load(t)
	if !ok {
		i, _ = cachedAbsorbers.LoadOrStore(t, &sync.Map{})
	}
	return i.(*sync.Map)
}

func getAbsorber(elemTyp reflect.Type, tag string, keys []string) *absorber {
	absorbers := getAbsorbersForType(elemTyp)

	compoundKey := tag + ":" + strings.Join(keys, "+")
	i, ok := absorbers.Load(compoundKey)
	if !ok {
		toPut := buildAbsorber(elemTyp, tag, keys)
		i, _ = absorbers.LoadOrStore(compoundKey, toPut)
	}
	return i.(*absorber)
}

// isIndirect returns true if t cannot be absorbed into directly.
// Returns true for Array, Can, Ptr, and Slice.
// Returns false for all scalars, structs, and maps.
func isIndirect(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Ptr, reflect.Slice:
		return true
	default:
		return false
	}
}

func elementType(t reflect.Type) reflect.Type {
	for isIndirect(t) {
		t = t.Elem()
	}
	return t
}

func buildAbsorber(elemTyp reflect.Type, tag string, keys []string) *absorber {
	a := &absorber{
		Elem: elemTyp,
		Keys: keys,
	}

	if a.Elem.Kind() == reflect.Struct {
		// TODO: Flip inner & outer loops, iterate fields & check struct tags.
		fields := make([]reflect.StructField, len(keys))
		for idx, key := range keys {
			if field, ok := a.Elem.FieldByName(key); ok {
				fields[idx] = field
			}
		}
		a.Fields = fields
	}

	return a
}

// element returns a new element of a's Elem type, constructed from the given values.
func (a *absorber) element(values []interface{}) reflect.Value {
	// Allocate a value for assignment
	dstVal := reflect.Indirect(reflect.New(a.Elem))

	switch a.Elem.Kind() {
	case reflect.Map:
		// Use the field names directly
		dstVal = reflect.MakeMapWithSize(a.Elem, len(values))
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
