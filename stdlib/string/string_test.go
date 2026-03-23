package string

import (
	"testing"

	"github.com/erickweyunga/sintax/object"
)

func s(v string) *object.StringObj { return &object.StringObj{Value: v} }

func assertStr(t *testing.T, name string, result object.Object, expected string) {
	t.Helper()
	str, ok := result.(*object.StringObj)
	if !ok {
		t.Fatalf("%s: expected str, got %T", name, result)
	}
	if str.Value != expected {
		t.Fatalf("%s: expected %q, got %q", name, expected, str.Value)
	}
}

func assertBool(t *testing.T, name string, result object.Object, expected bool) {
	t.Helper()
	b, ok := result.(*object.BoolObj)
	if !ok {
		t.Fatalf("%s: expected bool, got %T", name, result)
	}
	if b.Value != expected {
		t.Fatalf("%s: expected %v, got %v", name, expected, b.Value)
	}
}

func TestSplit(t *testing.T) {
	r, err := Split([]object.Object{s("hello-world"), s("-")})
	if err != nil {
		t.Fatal(err)
	}
	list := r.(*object.ListObj)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(list.Elements))
	}
	assertStr(t, "split[0]", list.Elements[0], "hello")
	assertStr(t, "split[1]", list.Elements[1], "world")
}

func TestJoin(t *testing.T) {
	list := &object.ListObj{Elements: []object.Object{s("a"), s("b"), s("c")}}
	r, err := Join([]object.Object{list, s(", ")})
	if err != nil {
		t.Fatal(err)
	}
	assertStr(t, "join", r, "a, b, c")
}

func TestReplace(t *testing.T) {
	r, _ := Replace([]object.Object{s("hello world"), s("world"), s("sintax")})
	assertStr(t, "replace", r, "hello sintax")
}

func TestTrim(t *testing.T) {
	r, _ := Trim([]object.Object{s("  hello  ")})
	assertStr(t, "trim", r, "hello")
}

func TestUpperLower(t *testing.T) {
	r, _ := Upper([]object.Object{s("hello")})
	assertStr(t, "upper", r, "HELLO")

	r, _ = Lower([]object.Object{s("HELLO")})
	assertStr(t, "lower", r, "hello")
}

func TestContains(t *testing.T) {
	r, _ := Contains([]object.Object{s("hello world"), s("world")})
	assertBool(t, "contains(true)", r, true)

	r, _ = Contains([]object.Object{s("hello world"), s("xyz")})
	assertBool(t, "contains(false)", r, false)
}

func TestStartsWithEndsWith(t *testing.T) {
	r, _ := StartsWith([]object.Object{s("hello"), s("hel")})
	assertBool(t, "starts_with", r, true)

	r, _ = EndsWith([]object.Object{s("hello"), s("llo")})
	assertBool(t, "ends_with", r, true)
}
