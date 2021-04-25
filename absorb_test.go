package absorb

import (
	"testing"
)

type testSource struct {
	i int
}

func (ts *testSource) Keys() []string {
	return []string{"One", "Two"}
}

func (ts *testSource) Next() (Values, error) {
	if ts.i < 5 {
		ts.i++
		return Values{"test", ts.i}, nil
	} else {
		return nil, nil
	}
}

type TestDst struct {
	One string
	Two int
}

func TestPointerToStruct(t *testing.T) {
	var src testSource
	var dst TestDst

	if err := Copy(&dst, "", &src); err != nil {
		t.Fatal(err)
	}
	t.Logf("Dst: %+v\n", dst)
	if expected := 5; dst.Two != expected {
		t.Fatal("dst.Two did not contain final value", "got", dst.Two, "expected", expected)
	}
}

func TestMap(t *testing.T) {
	var src testSource
	var dst map[string]interface{}

	if err := Copy(&dst, "", &src); err != nil {
		t.Fatal(err)
	}
	t.Logf("Dst: %+v\n", dst)
	if expected := 5; dst["Two"].(int) != expected {
		t.Fatal("dst.Two did not contain final value", "got", dst["Two"], "expected", expected)
	}
}
