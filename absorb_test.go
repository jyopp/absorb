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
func (ts testSource) Emit(into absorb.Absorber) error {
	into.Open("test", ts.i, "One", "Two")
	defer into.Close()

	for i := 0; i < ts.i; i++ {
		into.Absorb("test", i+1)
	}
	return nil
}

type TestDst struct {
	One string
	Two int
}

func TestStructPointer(t *testing.T) {
	src := testSource{i: 1}
	var dst TestDst

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	expect := TestDst{One: "test", Two: 1}
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
		expect := TestDst{One: "test", Two: idx + 1}
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
		expect := TestDst{One: "test", Two: idx}

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
	if expected := 1; dst["Two"].(int) != expected {
		t.Fatal("dst.Two did not contain final value", "got", dst["Two"], "expected", expected)
	}
}

func TestScalar(t *testing.T) {
	var dst int = -1

	abs := absorb.New(&dst)
	abs.Open("", 1)
	defer abs.Close()

	expect := 55
	if abs.Absorb(expect); dst != expect {
		t.Fatal("Expected", expect, "but got", dst)
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
