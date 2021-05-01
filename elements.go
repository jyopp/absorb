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

// absorb assigns the given values into the given element value.
//
// NOTE: For both efficiency and correctness, the returned value is of type
// reflect.PointerTo(a.Type) when possible.
func (a *elementBuilder) absorb(elem reflect.Value, values []interface{}) {
	if elem.Kind() == reflect.Ptr && elem.IsZero() {
		elem.Set(reflect.New(elem.Type().Elem()))
	}

	switch a.Type.Kind() {
	case reflect.Map:
		// Use the field names directly to make a map[string]T
		_assign(elem, reflect.MakeMapWithSize(a.Type, len(values)))
		elem = reflect.Indirect(elem)
		// Values are homogeneous, so just reuse one Value
		mapVal := reflect.Indirect(reflect.New(a.Type.Elem()))
		for idx, value := range values {
			key := reflect.ValueOf(a.Keys[idx])
			val := reflect.ValueOf(value)
			if val.IsValid() {
				_assign(mapVal, val)
				elem.SetMapIndex(key, mapVal)
			}
		}
	case reflect.Struct:
		// Ensure we are working with struct val when passed *struct
		elem = reflect.Indirect(elem)
		for idx, field := range a.Fields {
			val := reflect.ValueOf(values[idx])
			if val.IsValid() {
				f := elem.FieldByIndex(field.Index)
				_assign(f, val)
			}
		}
	default:
		switch len(values) {
		case 1:
			val := reflect.ValueOf(values[0])
			if t := val.Type(); t == elem.Type() {
				elem.Set(val)
			} else if t == reflect.PtrTo(a.Type) {
				elem.Set(reflect.Indirect(val))
			}
			_assign(elem, val)
		default:
			panic("cannot assign multiple columns to element of type " + a.Type.String())
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
		if dst.IsZero() {
			dst.Set(reflect.New(dstType.Elem()))
		}
		// Allocate a value and unwrap it for assignment
		dstType = dstType.Elem()
		dst = reflect.Indirect(dst)
		if srcType.AssignableTo(dstType) {
			dst.Set(src)
			return
		}
	}

	// Convert without checking convertability; We want panic on failure.
	dst.Set(src.Convert(dstType))
}
