package math

import (
	gomath "math"
	"testing"

	"github.com/erickweyunga/sintax/object"
)

func n(v float64) *object.NumberObj { return &object.NumberObj{Value: v} }

func assertNum(t *testing.T, name string, result object.Object, expected float64) {
	t.Helper()
	num, ok := result.(*object.NumberObj)
	if !ok {
		t.Fatalf("%s: expected num, got %T", name, result)
	}
	if gomath.Abs(num.Value-expected) > 0.0001 {
		t.Fatalf("%s: expected %g, got %g", name, expected, num.Value)
	}
}

func TestSqrt(t *testing.T) {
	r, err := Sqrt([]object.Object{n(16)})
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, "sqrt(16)", r, 4)
}

func TestAbs(t *testing.T) {
	r, err := Abs([]object.Object{n(-42)})
	if err != nil {
		t.Fatal(err)
	}
	assertNum(t, "abs(-42)", r, 42)
}

func TestFloorCeil(t *testing.T) {
	r, _ := Floor([]object.Object{n(3.7)})
	assertNum(t, "floor(3.7)", r, 3)

	r, _ = Ceil([]object.Object{n(3.2)})
	assertNum(t, "ceil(3.2)", r, 4)
}

func TestRound(t *testing.T) {
	r, _ := Round([]object.Object{n(3.5)})
	assertNum(t, "round(3.5)", r, 4)
}

func TestMinMax(t *testing.T) {
	r, _ := Min([]object.Object{n(5), n(10)})
	assertNum(t, "min(5,10)", r, 5)

	r, _ = Max([]object.Object{n(5), n(10)})
	assertNum(t, "max(5,10)", r, 10)
}

func TestPi(t *testing.T) {
	r, _ := Pi(nil)
	assertNum(t, "pi()", r, gomath.Pi)
}

func TestWrongArgs(t *testing.T) {
	_, err := Sqrt([]object.Object{})
	if err == nil {
		t.Fatal("expected error for sqrt() with no args")
	}

	_, err = Sqrt([]object.Object{&object.StringObj{Value: "bad"}})
	if err == nil {
		t.Fatal("expected error for sqrt(string)")
	}
}
