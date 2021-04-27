package absorb

import (
	"reflect"
)

// Absorbable defines the interface for types that may fill Absorbers with values.
type Absorbable interface {
	// Emit places the entire contents of the receiver into the provided Absorber.
	//
	// Implementations must Open the Absorber and call Absorb for each value.
	Emit(into Absorber) error
}

type Absorber interface {
	// Open configures the Absorber to accept elements using the given set of keys.
	// The given tagname (such as "mydb") is preferred when mapping keys to struct fields.
	// Count is a hint about the number of items this Absorber can produce. If the number
	// of items is unknown, pass -1.
	//
	// If no keys are provided, Absorb may be called at most once, with a single value.
	//
	// Panics if count is greater than the absorber's maximum size.
	Open(tag string, count int, keys ...string)
	// Absorb creates an output element from the given values and adds it to the output.
	//
	// If the output type is a channel, this method may block.
	// If the output type is array (not slice), panics on overflow.
	Absorb(values ...interface{})
	// Close releases internal resources and assigns the output when relevant.
	Close()
}

/*
	Absorb absorbs all source values into a new Absorber for dst.
	Equivalent to src.Emit(absorb.New(dst)).

	Examples:
	  var mySlice []structType
	  err := absorb.Absorb(&mySlice, dataSource)
	  structChan := make(chan structType)
	  err = absorb.Absorb(structChan, rowReader)
*/
func Absorb(dst interface{}, src Absorbable) error {
	return src.Emit(New(dst))
}

// Create a new Absorber that writes elements of the corresponding type into dst.
// Panics if dst is not an assignable reference or a channel.
func New(dst interface{}) Absorber {
	// Consider the types:
	// DstVal           ContainerVal   Elem
	// *[]struct        []struct       struct
	// chan struct      <---           struct
	// *struct          <---           struct
	// *int             <---           int
	// *[10]map[s]i     [10]map[s]i    map[s]i

	// Known Issues:
	// *[]T expects one T per loop iteration; Absorb(T1, T2, T3) will panic.
	// The best workaround is to not use absorb for single-valued iteration of this type.
	// If absorb is required, create an Absorber that just stores the arguments to Absorb().

	dstVal := reflect.ValueOf(dst)
	var setVal reflect.Value

	switch dstVal.Kind() {
	case reflect.Ptr:
		// The default case; We'll set dstVal.Elem() when accepting values.
		setVal = dstVal.Elem()
	case reflect.Chan:
		if dstVal.Type().ChanDir() == reflect.RecvDir {
			panic("cannot absorb into receive-only channel of type " + dstVal.Type().String())
		}
		// It is correct to pass Channels directly; Skip a level of indirection.
		setVal = dstVal
	default:
		panic("cannot absorb into (non-ptr, non-chan) " + dstVal.Type().String())
	}

	return &absorberImpl{
		dst:    dst,
		setVal: setVal,
	}
}

type absorberImpl struct {
	dst     interface{}
	idx     int
	setVal  reflect.Value
	builder *elementBuilder
	unwrap  bool
}

func (a *absorberImpl) Open(tag string, count int, keys ...string) {
	// Examine setVal to get element type and descend into its type structure as needed.
	elemTyp := a.setVal.Type()
	switch elemTyp.Kind() {
	case reflect.Array:
		if count > elemTyp.Len() {
			panic("cannot absorb: would exceed capacity of " + elemTyp.String())
		}
		// one key => array of single values; no keys => single value of type array
		if len(keys) > 0 {
			elemTyp = elemTyp.Elem()
		}
	case reflect.Slice:
		// one key => slice of values; no keys => single value of type slice
		if len(keys) > 0 {
			elemTyp = elemTyp.Elem()
		}
	case reflect.Chan:
		elemTyp = elemTyp.Elem()
	default:
		if count > 1 {
			panic("cannot absorb multiple values into single-valued type " + elemTyp.String())
		}
	}

	// Reset the index; An absorber could be re-used.
	a.idx = 0

	if elemTyp.Kind() == reflect.Ptr {
		// If we ended on a pointer type, dereference it one more time
		elemTyp = elemTyp.Elem()
		a.unwrap = false
	} else {
		// Else indicate that we DON'T have a pointer, so elements may need to be unwrapped before accepting them
		a.unwrap = true
	}
	a.builder = getBuilder(elemTyp, tag, keys)
}

func (a *absorberImpl) Absorb(values ...interface{}) {
	elem := a.builder.element(values)
	if a.unwrap {
		elem = reflect.Indirect(elem)
	}
	idx := a.idx
	if idx > 0 && len(a.builder.Keys) == 0 {
		panic("cannot accept multiple items when no keys were provided")
	}
	accept(a.setVal, elem, idx)
	a.idx = idx + 1
}

func accept(into, elem reflect.Value, idx int) {
	// Append an element to an output value.
	switch into.Kind() {
	case reflect.Chan:
		into.Send(elem)
	case reflect.Slice:
		if into.Type() == elem.Type() {
			// Necessary to support &[]byte
			into.Set(elem)
		} else {
			into.Set(reflect.Append(into, elem))
		}
	case reflect.Array:
		into.Index(idx).Set(elem)
	case reflect.Ptr:
		if elem.Kind() == reflect.Ptr {
			// Set the pointer directly, panic on type mismatch
			into.Set(elem)
		} else {
			// Store value in a new pointer
			into.Set(reflect.New(into.Type().Elem()))
			into = reflect.Indirect(into)
			into.Set(elem)
		}
	default:
		into.Set(elem)
	}
}

func (a *absorberImpl) Close() {
	// Not strictly necessary, but the Open/Close pattern is clear and useful.
	a.builder = nil
}
