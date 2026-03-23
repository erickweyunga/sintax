package json

import (
	"testing"

	"github.com/erickweyunga/sintax/object"
)

func s(v string) *object.StringObj { return &object.StringObj{Value: v} }

func TestParseNumber(t *testing.T) {
	r, err := Parse([]object.Object{s("42")})
	if err != nil {
		t.Fatal(err)
	}
	n := r.(*object.NumberObj)
	if n.Value != 42 {
		t.Fatalf("expected 42, got %g", n.Value)
	}
}

func TestParseString(t *testing.T) {
	r, err := Parse([]object.Object{s(`"hello"`)})
	if err != nil {
		t.Fatal(err)
	}
	if r.(*object.StringObj).Value != "hello" {
		t.Fatalf("expected 'hello', got %s", r.Inspect())
	}
}

func TestParseBool(t *testing.T) {
	r, _ := Parse([]object.Object{s("true")})
	if !r.(*object.BoolObj).Value {
		t.Fatal("expected true")
	}
	r, _ = Parse([]object.Object{s("false")})
	if r.(*object.BoolObj).Value {
		t.Fatal("expected false")
	}
}

func TestParseNull(t *testing.T) {
	r, _ := Parse([]object.Object{s("null")})
	if _, ok := r.(*object.NullObj); !ok {
		t.Fatalf("expected null, got %T", r)
	}
}

func TestParseArray(t *testing.T) {
	r, err := Parse([]object.Object{s(`[1, 2, 3]`)})
	if err != nil {
		t.Fatal(err)
	}
	list := r.(*object.ListObj)
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(list.Elements))
	}
}

func TestParseObject(t *testing.T) {
	r, err := Parse([]object.Object{s(`{"name": "Eric", "age": 25}`)})
	if err != nil {
		t.Fatal(err)
	}
	dict := r.(*object.DictObj)
	if len(dict.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(dict.Pairs))
	}
	if dict.Pairs["name"].(*object.StringObj).Value != "Eric" {
		t.Fatalf("expected name=Eric")
	}
}

func TestParseNested(t *testing.T) {
	r, err := Parse([]object.Object{s(`{"users": [{"name": "Eric"}, {"name": "John"}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	dict := r.(*object.DictObj)
	users := dict.Pairs["users"].(*object.ListObj)
	if len(users.Elements) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users.Elements))
	}
}

func TestStringify(t *testing.T) {
	dict := &object.DictObj{
		Pairs: map[string]object.Object{
			"name": &object.StringObj{Value: "Eric"},
			"age":  &object.NumberObj{Value: 25},
		},
		Keys: []string{"name", "age"},
	}
	r, err := Stringify([]object.Object{dict})
	if err != nil {
		t.Fatal(err)
	}
	s := r.(*object.StringObj).Value
	if s != `{"name":"Eric","age":25}` {
		t.Fatalf("unexpected JSON: %s", s)
	}
}

func TestStringifyList(t *testing.T) {
	list := &object.ListObj{Elements: []object.Object{
		&object.NumberObj{Value: 1},
		&object.NumberObj{Value: 2},
		&object.NumberObj{Value: 3},
	}}
	r, _ := Stringify([]object.Object{list})
	if r.(*object.StringObj).Value != "[1,2,3]" {
		t.Fatalf("unexpected: %s", r.Inspect())
	}
}

func TestPretty(t *testing.T) {
	dict := &object.DictObj{
		Pairs: map[string]object.Object{
			"name": &object.StringObj{Value: "Eric"},
		},
		Keys: []string{"name"},
	}
	r, err := Pretty([]object.Object{dict})
	if err != nil {
		t.Fatal(err)
	}
	s := r.(*object.StringObj).Value
	if s != "{\n  \"name\": \"Eric\"\n}" {
		t.Fatalf("unexpected pretty JSON: %s", s)
	}
}

func TestRoundTrip(t *testing.T) {
	input := `{"items":[1,2,3],"active":true,"name":"test"}`
	parsed, _ := Parse([]object.Object{s(input)})
	stringified, _ := Stringify([]object.Object{parsed})
	reparsed, _ := Parse([]object.Object{stringified.(*object.StringObj)})

	dict := reparsed.(*object.DictObj)
	if dict.Pairs["active"].(*object.BoolObj).Value != true {
		t.Fatal("round trip failed for bool")
	}
	items := dict.Pairs["items"].(*object.ListObj)
	if len(items.Elements) != 3 {
		t.Fatal("round trip failed for list")
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse([]object.Object{s("{invalid}")})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
