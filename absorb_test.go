package absorb_test

import (
	"reflect"
	"testing"

	"github.com/jyopp/absorb"
)

type testSource struct {
	i int
}

// testSource implements absorb.Absorbable
// It emits its keys in the tag namespace "test"
func (ts testSource) Emit(into absorb.Absorber) error {
	// Pass -1 for expected size, to ensure the default path gets checked
	into.Open("test", -1, "Name", "Aliased")
	defer into.Close()

	for i := 0; i < ts.i; i++ {
		into.Absorb("test", i+1)
	}
	return nil
}

type TestDst struct {
	Name   string
	Actual int `test:"Aliased"`
}

func TestStructPointer(t *testing.T) {
	src := testSource{i: 1}
	var dst TestDst

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	expect := TestDst{Name: "test", Actual: 1}
	if dst != expect {
		t.Fatalf("Expected %+v, got %+v", expect, dst)
	}
}

func TestStructSlice(t *testing.T) {
	src := testSource{i: 5}
	var dst []TestDst

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	if len(dst) != src.i {
		t.Fatalf("Expected %d structs, got %d", src.i, len(dst))
	}
	for idx := range dst {
		expect := TestDst{Name: "test", Actual: idx + 1}
		if dst[idx] != expect {
			t.Fatalf("Expected %+v, got %+v", expect, dst[idx])
		}
	}
}

func TestStructChannel(t *testing.T) {
	// Unbuffered channel of struct
	dst := make(chan TestDst)

	go func() {
		src := testSource{i: 5}
		absorb.Absorb(dst, src)
		close(dst)
	}()

	idx := 0
	for received := range dst {
		idx++
		expect := TestDst{Name: "test", Actual: idx}

		if received != expect {
			t.Fatalf("Expected %+v, got %+v", expect, received)
		}
	}
}

func TestMap(t *testing.T) {
	src := testSource{i: 1}
	var dst map[string]interface{}

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	t.Logf("Dst: %+v\n", dst)
	if expected := 1; dst["Aliased"].(int) != expected {
		t.Fatal("dst[Aliased] did not contain final value", "got", dst["Aliased"], "expected", expected)
	}
}

func TestMapSlice(t *testing.T) {
	src := testSource{i: 5}
	var dst []map[string]interface{}

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	if len(dst) != src.i {
		t.Fatalf("Expected %d structs, got %d", src.i, len(dst))
	}
	for idx := range dst {
		expect := map[string]interface{}{"Name": "test", "Aliased": idx + 1}
		if !reflect.DeepEqual(dst[idx], expect) {
			t.Fatalf("Expected %+v, got %+v", expect, dst[idx])
		}
	}
}

func TestInt(t *testing.T) {
	var dst int = -1

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := 55
	if abs.Absorb(expect); dst != expect {
		t.Fatal("Expected", expect, "but got", dst)
	}
}

func TestIntPointer(t *testing.T) {
	var dst *int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := 55
	if abs.Absorb(expect); dst == nil {
		t.Fatal("Failed to absorb int to *int")
	} else if *dst != expect {
		t.Fatal("Expected", expect, "but got", *dst)
	}
}

func TestIntFromPointer(t *testing.T) {
	var dst int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := 55
	if abs.Absorb(&expect); dst != expect {
		t.Fatal("Expected", expect, "but got", dst)
	}
}

func TestIntFromPointerConversion(t *testing.T) {
	var dst int64

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := int32(55)
	if abs.Absorb(&expect); dst != int64(expect) {
		t.Fatal("Expected", expect, "but got", dst)
	}
}

func TestPointerFromPointer(t *testing.T) {
	var dst *int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := 55
	if abs.Absorb(&expect); dst != &expect {
		t.Fatal("Expected", &expect, "but got", &dst)
	}
}

func TestPointerFromPointerConversion(t *testing.T) {
	var dst *int64

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := int32(55)
	if abs.Absorb(&expect); *dst != int64(expect) {
		t.Fatal("Expected", &expect, "but got", &dst)
	}
}

func TestPointerFieldFromPointerConversion(t *testing.T) {
	type StructWithPointerMember struct {
		Value *string
	}
	var dst StructWithPointerMember

	abs := absorb.New(&dst)
	abs.Open("", 1, "Value")
	defer abs.Close()

	strVal := "Test String"
	if abs.Absorb(&strVal); dst.Value != &strVal {
		t.Fatal("Pointers to strings do not match", dst.Value, &strVal)
	}
}

func TestPointerFieldFromConcreteConversion(t *testing.T) {
	type StructWithPointerMember struct {
		Value *string
	}
	var dst StructWithPointerMember

	abs := absorb.New(&dst)
	abs.Open("", 1, "Value")
	defer abs.Close()

	strVal := "Test String"
	if abs.Absorb(strVal); dst.Value == nil {
		t.Fatal("Assigning pointer field from concrete string failed")
	} else if *dst.Value != strVal {
		t.Fatal("Copied string does not match input", *dst.Value, strVal)
	}
}

func TestMapPointer(t *testing.T) {
	var dst *map[string]int

	abs := absorb.New(&dst)
	abs.Open("", 1, "One", "Two")
	defer abs.Close()

	// Intentionally write a partially-valid row.
	// The nil / invalid value must be omitted from the resulting map
	abs.Absorb(55, nil)

	if dst == nil {
		t.Fatal("Failed to absorb values into pointer-to-map")
	}
	if twoVal, ok := (*dst)["Two"]; ok {
		t.Fatal("Map contains Two =", twoVal, "for nil source value")
	}
	if oneVal, ok := (*dst)["One"]; !ok || oneVal != 55 {
		t.Fatal("Map contains One =", oneVal, "but expected 55")
	}
}

func TestSlice(t *testing.T) {
	var dst []int

	abs := absorb.New(&dst)
	abs.Open("", 4)
	defer abs.Close()

	expect := []int{34, 55, 22, 1}
	for _, i := range expect {
		abs.Absorb(i)
	}
	if !reflect.DeepEqual(dst, expect) {
		t.Fatal("Expected", expect, "but got", dst)
	}
}

func TestArray(t *testing.T) {
	var dst [4]int

	abs := absorb.New(&dst)
	abs.Open("", 4)
	defer abs.Close()

	expect := []int{34, 55, 22, 1}
	for _, i := range expect {
		abs.Absorb(i)
	}
	if !reflect.DeepEqual(dst[:], expect) {
		t.Fatal("Expected", expect, "but got", dst)
	}
}

func TestUnassignablePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for unassignable destination")
		}
	}()

	var dst int

	// Should panic if configured with a non-assignable type
	_ = absorb.New(dst)
}

func TestOvercountPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for out-of-bounds absorption")
		}
	}()

	var dst int

	abs := absorb.New(&dst)
	// Open should panic if dst can't hold the expected count.
	abs.Open("", 2)
}

func TestArrayOvercountPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for out-of-bounds absorption")
		}
	}()

	var dst [5]int

	abs := absorb.New(&dst)
	// Open should panic if dst can't hold the expected count.
	abs.Open("", 7)
}

func TestOverflowPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for multiple writes to single pointer")
		}
	}()

	var dst int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	abs.Absorb(1)
	// Absorb should panic if a second item is written.
	abs.Absorb(2)
}

func TestPointerMultivaluePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for multi-valued write to single pointer")
		}
	}()

	var dst int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	// Absorb must panic if trying to put two values into a single pointer.
	abs.Absorb(1, 3)
}

func TestEmptyRowPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for zero-valued write to single pointer")
		}
	}()

	var dst int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	// Absorb must panic if trying to put two values into a single pointer.
	abs.Absorb()
}

func TestPointerOverflowPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for multiple writes to single pointer")
		}
	}()

	var dst *int

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	abs.Absorb(1)
	// Absorb should panic if a second item is written.
	abs.Absorb(2)
}

func TestReceiveOnlyPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Failed to panic for receive-only channel")
		}
	}()

	sendRcv := make(chan TestDst)
	var rcvOnly <-chan TestDst = sendRcv

	// absorb.New should panic if channel cannot be written to.
	_ = absorb.New(rcvOnly)
}
