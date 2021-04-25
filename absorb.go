package absorb

import (
	"reflect"
)

type Values []interface{}

type Countable interface {
	Count() int
}

// Absorbable defines an iterable interface where Keys are retrieved once, and
// values are iterated repeatedly.
type Absorbable interface {
	// Keys must return the ordered source keys for absorption.
	// If Keys returns nil, the zeroth value in NextValues() will be absorbed directly.
	Keys() []string
	// Next must return the constituent values of the next absorbable type.
	// When no absorbable values remain, NextValues must return (nil, nil) to end iteration.
	//
	// Values must be returned in the same order as Keys, or if Keys is nil,
	// the zeroth value will be absorbed directly.
	Next() (Values, error)
}

// Copy copies values from the given Absorbable into dst.
// If tag is nonempty, dst must be a struct, chan struct, or slice of struct.
// The given tag is used to match Absorbable Pairs to the output object.
// Returns the first error reported by src.
// Panics if the pairs produced by src cannot be fully mapped to dst.
func Copy(dst interface{}, tag string, src Absorbable) error {
	outVal := reflect.ValueOf(dst)
	// To be assignable, dstType must be indirect (have a settable element)
	if !isIndirect(outVal.Type()) || !outVal.Elem().CanSet() {
		return &reflect.ValueError{
			Method: "Set",
			Kind:   outVal.Kind(),
		}
	}

	keys := src.Keys()
	var count int
	if countable, ok := src.(Countable); ok {
		count = countable.Count()
	} else {
		count = -1
	}

	absorber := getAbsorber(outVal.Type(), "", keys)

	print("Count", count)

	vals, err := src.Next()
	for ; vals != nil; vals, err = src.Next() {
		elem := absorber.element(vals)
		println(elem.String())
		outVal.Elem().Set(elem)
	}
	if err != nil {
		return err
	}
	// If count is zero, try to return null else return an error.
	// If count is > 1 and result is not a slice or channel, return an error.

	// Find outType, the underlying type of dst
	// Get or build a cached mapping that can assign keys to outType

	return nil
}
