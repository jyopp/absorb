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
		mappedFields := make(map[string]reflect.StructField)
		for i := 0; i < elemTyp.NumField(); i++ {
			field := elemTyp.Field(i)
			if tagVal, ok := field.Tag.Lookup(tag); ok {
				// If a field has a matching struct tag, ONLY the tag is used.
				// If the tag is explicitly empty, the field is excluded.
				if tagVal != "" {
					mappedFields[tagVal] = field
				}
			} else {
				// Use the field's name and its lowercased name for matching.
				mappedFields[field.Name] = field
				lowered := strings.ToLower(field.Name)
				// Lowercased names are set conditionally, to avoid clobbering tags & other fields
				if _, ok := mappedFields[lowered]; !ok {
					mappedFields[lowered] = field
				}
			}
		}

		fields := make([]reflect.StructField, len(keys))
		for idx, key := range keys {
			if field, ok := mappedFields[key]; ok {
				fields[idx] = field
			} else {
				// Fall back to case-insensitive match
				fields[idx] = mappedFields[strings.ToLower(key)]
			}
		}
		a.Fields = fields
	}

	return a
}

// element returns a new element of a's Elem type, constructed from the given values.
//
// NOTE: For both efficiency and correctness, the returned value is of type
// reflect.PointerTo(a.Type) when possible.
func (a *elementBuilder) element(values []interface{}) reflect.Value {
	switch a.Type.Kind() {
	case reflect.Map:
		// Use the field names directly
		dstVal := reflect.MakeMapWithSize(a.Type, len(values))
		// Values are homogeneous, so just reuse one Value
		mapVal := reflect.Indirect(reflect.New(dstVal.Type().Elem()))
		for idx, value := range values {
			key := reflect.ValueOf(a.Keys[idx])
			val := reflect.ValueOf(value)
			if val.IsValid() {
				_assign(mapVal, val)
				dstVal.SetMapIndex(key, mapVal)
			}
		}
		return dstVal
	case reflect.Struct:
		ptrVal := reflect.New(a.Type)
		dstVal := reflect.Indirect(ptrVal)
		for idx, field := range a.Fields {
			val := reflect.ValueOf(values[idx])
			if val.IsValid() {
				f := dstVal.FieldByIndex(field.Index)
				_assign(f, val)
			}
		}
		return ptrVal
	default:
		switch len(values) {
		case 0:
			return reflect.Value{}
		case 1:
			val := reflect.ValueOf(values[0])
			if t := val.Type(); t == a.Type || t == reflect.PtrTo(a.Type) {
				return val
			}
			dstVal := reflect.New(a.Type)
			_assign(reflect.Indirect(dstVal), val)
			return dstVal
		default:
			panic("cannot assign multiple values to element of type " + a.Type.String())
		}
	}
}

func _assign(dst, src reflect.Value) {
	dstType, srcType := dst.Type(), src.Type()

	if dstType == srcType || srcType.AssignableTo(dstType) {
		// Happy Path
		dst.Set(src)
		return
	}

	// If one or both values is a pointer, the unwrapped types may be assignable or convertible.
	if srcType.Kind() == reflect.Ptr {
		// Reassign src to its contained value.
		srcType = srcType.Elem()
		src = reflect.Indirect(src)
	}
	// For struct and map fields (top-level values handled elsewhere)
	// Handle concrete-to-pointer and pointer-to-pointer conversions
	if dstType.Kind() == reflect.Ptr {
		// Allocate a value and unwrap it for assignment
		dstType = dstType.Elem()
		dst.Set(reflect.New(dstType))
		dst = reflect.Indirect(dst)
		if srcType.AssignableTo(dstType) {
			dst.Set(src)
			return
		}
	}

	// Convert without checking convertability; We want panic on failure.
	dst.Set(src.Convert(dstType))
}
