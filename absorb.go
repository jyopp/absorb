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
	// Returns the maximum number of times Absorb may be called, which is always greater than zero.
	// For channel and slice types, returns INT_MAX. For struct and map pointers, returns 1.
	// For array pointers, a fixed number is returned; Absorb panics if an array overflows.
	Open(tag string, count int, keys ...string) int
	// Absorb creates an output element from the given values and adds it to the output.
	//
	// If the output type is a channel, this method may block.
	// If the output type is array (not slice), panics on overflow.
	Absorb(values ...interface{})
	// Close releases internal resources and assigns the output when relevant.
	Close()
}

// Create a new Absorber that writes elements of the corresponding type into dst.
func New(dst interface{}) Absorber {
	return &absorberImpl{
		dst: dst,
	}
}

func Absorb(dst interface{}, src Absorbable) error {
	return src.Emit(New(dst))
}

type absorberImpl struct {
	dst          interface{}
	elemAbsorber *absorber
}

func (a *absorberImpl) Open(tag string, count int, keys ...string) int {
	dstTyp := reflect.TypeOf(a.dst)
	elemTyp := elementType(dstTyp)
	a.elemAbsorber = getAbsorber(elemTyp, tag, keys)

	// TODO: Recursively inspect dst's type to determine real count to return.
	return count
}

func (a *absorberImpl) Absorb(values ...interface{}) {
	elem := a.elemAbsorber.element(values)
	a.accept(elem)
}

func (a *absorberImpl) accept(elem reflect.Value) {
	// Actually append the absorbed value to the output.
	// TODO: Support all the crazy stuff. This only works with map & struct pointers.
	reflect.ValueOf(a.dst).Elem().Set(elem)
}

func (a *absorberImpl) Close() {
	// Not strictly necessary, but the Open/Close pattern is clear and useful.
	a.elemAbsorber = nil
}
