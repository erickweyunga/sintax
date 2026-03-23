package json

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/stdlib/types"
)

// Load returns the json module.
func Load() *types.Module {
	return &types.Module{
		Name: "json",
		Desc: "JSON parse and stringify",
		Funcs: map[string]types.StdFn{
			"parse":     Parse,
			"stringify":  Stringify,
			"pretty":    Pretty,
		},
		Info: []types.FuncInfo{
			{Name: "parse(str)", Desc: "Parse JSON string to value"},
			{Name: "stringify(value)", Desc: "Convert value to JSON string"},
			{Name: "pretty(value)", Desc: "Convert value to pretty JSON string"},
		},
	}
}

// Parse converts a JSON string into a Sintax value.
func Parse(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: json/parse() requires 1 argument")
	}
	s, ok := args[0].(*object.StringObj)
	if !ok {
		return nil, fmt.Errorf("Error: json/parse() requires a str")
	}

	// Use Decoder to preserve key order
	dec := json.NewDecoder(strings.NewReader(s.Value))
	dec.UseNumber()
	result, err := decodeValue(dec)
	if err != nil {
		return nil, fmt.Errorf("Error: invalid JSON: %s", err.Error())
	}
	return result, nil
}

// Stringify converts a Sintax value to a compact JSON string.
func Stringify(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: json/stringify() requires 1 argument")
	}
	data, err := marshalSintax(args[0])
	if err != nil {
		return nil, fmt.Errorf("Error: cannot stringify to JSON")
	}
	return &object.StringObj{Value: string(data)}, nil
}

// Pretty converts a Sintax value to an indented JSON string.
func Pretty(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: json/pretty() requires 1 argument")
	}
	data, err := marshalPretty(args[0])
	if err != nil {
		return nil, fmt.Errorf("Error: cannot stringify to JSON")
	}
	return &object.StringObj{Value: string(data)}, nil
}

// decodeValue reads a single JSON value from the decoder, preserving object key order.
func decodeValue(dec *json.Decoder) (object.Object, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			return decodeObject(dec)
		case '[':
			return decodeArray(dec)
		}
	case json.Number:
		f, _ := v.Float64()
		return &object.NumberObj{Value: f}, nil
	case string:
		return &object.StringObj{Value: v}, nil
	case bool:
		return &object.BoolObj{Value: v}, nil
	case nil:
		return object.Null, nil
	}
	return object.Null, nil
}

func decodeObject(dec *json.Decoder) (object.Object, error) {
	pairs := make(map[string]object.Object)
	var keys []string

	for dec.More() {
		// Read key
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key := tok.(string)
		keys = append(keys, key)

		// Read value
		val, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		pairs[key] = val
	}

	// Consume closing }
	dec.Token()
	return &object.DictObj{Pairs: pairs, Keys: keys}, nil
}

func decodeArray(dec *json.Decoder) (object.Object, error) {
	var elements []object.Object
	for dec.More() {
		val, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}
	// Consume closing ]
	dec.Token()
	return &object.ListObj{Elements: elements}, nil
}

// marshalSintax converts a Sintax object to JSON bytes, preserving dict key order.
func marshalSintax(obj object.Object) ([]byte, error) {
	return marshalIndent(obj, "", "")
}

func marshalPretty(obj object.Object) ([]byte, error) {
	return marshalIndent(obj, "", "  ")
}

func marshalIndent(obj object.Object, prefix, indent string) ([]byte, error) {
	var buf strings.Builder
	writeJSON(&buf, obj, prefix, indent, 0)
	return []byte(buf.String()), nil
}

func writeJSON(buf *strings.Builder, obj object.Object, prefix, indent string, depth int) {
	switch v := obj.(type) {
	case *object.NumberObj:
		if v.Value == float64(int64(v.Value)) {
			fmt.Fprintf(buf, "%d", int64(v.Value))
		} else {
			fmt.Fprintf(buf, "%g", v.Value)
		}
	case *object.StringObj:
		data, _ := json.Marshal(v.Value)
		buf.Write(data)
	case *object.BoolObj:
		if v.Value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case *object.NullObj:
		buf.WriteString("null")
	case *object.ListObj:
		buf.WriteByte('[')
		for i, el := range v.Elements {
			if i > 0 {
				buf.WriteByte(',')
			}
			if indent != "" {
				buf.WriteByte('\n')
				buf.WriteString(prefix)
				for j := 0; j <= depth; j++ {
					buf.WriteString(indent)
				}
			}
			writeJSON(buf, el, prefix, indent, depth+1)
		}
		if indent != "" && len(v.Elements) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(prefix)
			for j := 0; j < depth; j++ {
				buf.WriteString(indent)
			}
		}
		buf.WriteByte(']')
	case *object.DictObj:
		buf.WriteByte('{')
		for i, k := range v.Keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if indent != "" {
				buf.WriteByte('\n')
				buf.WriteString(prefix)
				for j := 0; j <= depth; j++ {
					buf.WriteString(indent)
				}
			}
			keyData, _ := json.Marshal(k)
			buf.Write(keyData)
			buf.WriteByte(':')
			if indent != "" {
				buf.WriteByte(' ')
			}
			writeJSON(buf, v.Pairs[k], prefix, indent, depth+1)
		}
		if indent != "" && len(v.Keys) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(prefix)
			for j := 0; j < depth; j++ {
				buf.WriteString(indent)
			}
		}
		buf.WriteByte('}')
	default:
		buf.WriteString("null")
	}
}
