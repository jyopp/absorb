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
	setVal := a.setVal

	// Examine setVal to get element type and, when appropriate, allocate a container.
	var elemTyp reflect.Type
	var noUnwrap bool
	switch setVal.Kind() {
	case reflect.Ptr:
		// Tell element builder to return a pointer IFF types match.
		elemTyp = setVal.Type().Elem()
		noUnwrap = true
	case reflect.Array:
		if count > setVal.Type().Len() {
			panic("cannot absorb: would exceed capacity of " + setVal.Type().String())
		}
		elemTyp = setVal.Type().Elem()
	case reflect.Slice:
		elemTyp = setVal.Type().Elem()
		// Replace setVal with a new slice with reserved capacity.
		cap := count
		if cap < 0 {
			cap = 25
		}
		setVal.Set(reflect.MakeSlice(setVal.Type(), 0, cap))
	case reflect.Chan:
		elemTyp = setVal.Type().Elem()
		// If this is a channel of pointers-to-stuff, mark it appropriately.
		if elemTyp.Kind() == reflect.Ptr {
			elemTyp = elemTyp.Elem()
			noUnwrap = true
		}
	default:
		if count > 1 {
			panic("Too many items for scalar type " + setVal.Type().String())
		}
		elemTyp = setVal.Type()
	}

	// Now reset the absorber so it can start absorbing values.
	a.idx = 0
	a.builder = getBuilder(elemTyp, tag, keys)
	a.unwrap = (elemTyp.Kind() != reflect.Ptr) && !noUnwrap
}

func (a *absorberImpl) Absorb(values ...interface{}) {
	elem := a.builder.element(values)
	if a.unwrap {
		elem = reflect.Indirect(elem)
	}
	accept(a.setVal, elem, a.idx)
	a.idx++
}

func accept(into, elem reflect.Value, idx int) {
	// Append an element to an output value.
	switch into.Kind() {
	case reflect.Chan:
		into.Send(elem)
	case reflect.Slice:
		into.Set(reflect.Append(into, elem))
	case reflect.Array:
		into.Index(idx).Set(elem)
	case reflect.Ptr:
		if idx > 0 {
			panic("cannot absorb multiple items into " + into.Type().String())
		}
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
		if idx > 0 {
			panic("cannot absorb multiple items into " + into.Type().String())
		}
		into.Set(elem)
	}
}

func (a *absorberImpl) Close() {
	// Not strictly necessary, but the Open/Close pattern is clear and useful.
	a.builder = nil
}
