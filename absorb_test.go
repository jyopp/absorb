package absorb_test

import (
	"testing"

	"github.com/jyopp/absorb"
)

type testSource struct {
	i int
}

// testSource implements absorb.Absorbable
func (ts testSource) Emit(into absorb.Absorber) error {
	count := into.Open("test", ts.i, "One", "Two")
	defer into.Close()

	for i := 0; i < count; i++ {
		into.Absorb("test", i+1)
	}
	return nil
}

type TestDst struct {
	One string
	Two int
}

func TestPointerToStruct(t *testing.T) {
	src := testSource{i: 5}
	var dst TestDst

	if err := absorb.Absorb(&dst, src); err != nil {
		t.Fatal(err)
	}
	t.Logf("Dst: %+v\n", dst)
	if expected := 5; dst.Two != expected {
		t.Fatal("dst.Two did not contain final value", "got", dst.Two, "expected", expected)
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
