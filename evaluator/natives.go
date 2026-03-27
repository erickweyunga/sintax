package evaluator

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

func init() {
	natives := map[string]BuiltinFn{
		// Math
		"__native_sqrt":   nativeMath1(math.Sqrt),
		"__native_sin":    nativeMath1(math.Sin),
		"__native_cos":    nativeMath1(math.Cos),
		"__native_tan":    nativeMath1(math.Tan),
		"__native_asin":   nativeMath1(math.Asin),
		"__native_acos":   nativeMath1(math.Acos),
		"__native_atan":   nativeMath1(math.Atan),
		"__native_log":    nativeMath1(math.Log),
		"__native_log2":   nativeMath1(math.Log2),
		"__native_log10":  nativeMath1(math.Log10),
		"__native_exp":    nativeMath1(math.Exp),
		"__native_floor":  nativeMath1(math.Floor),
		"__native_ceil":   nativeMath1(math.Ceil),
		"__native_round":  nativeMath1(math.Round),
		"__native_cbrt":   nativeMath1(math.Cbrt),
		"__native_pow":    nativeMath2(math.Pow),
		"__native_random": nativeRandom,
		// String
		"__native_upper":          nativeUpper,
		"__native_lower":          nativeLower,
		"__native_split":          nativeSplit,
		"__native_replace":        nativeReplace,
		"__native_trim":           nativeTrim,
		"__native_char_code":      nativeCharCode,
		"__native_from_char_code": nativeFromCharCode,
		"__native_str_reverse":    nativeStrReverse,
		"__native_str_repeat":     nativeStrRepeat,
		"__native_index_of":       nativeIndexOf,
		"__native_slice":          nativeSlice,
		// List
		"__native_list_concat":  nativeListConcat,
		"__native_list_insert":  nativeListInsert,
		"__native_list_reverse": nativeListReverse,
		// Dict
		"__native_dict_delete": nativeDictDelete,
		"__native_dict_merge":  nativeDictMerge,
		// OS
		"__native_read_file":   nativeReadFile,
		"__native_write_file":  nativeWriteFile,
		"__native_file_exists": nativeFileExists,
		"__native_delete_file": nativeDeleteFile,
		"__native_cwd":         nativeCwd,
		"__native_getenv":      nativeGetenv,
		"__native_exec":        nativeExec,
		"__native_time":        nativeTime,
		"__native_sleep":       nativeSleep,
		"__native_exit":        nativeExit,
		"__native_format_time": nativeFormatTime,
		"__native_rename":      nativeRename,
		// JSON
		"__native_json_parse":     nativeJsonParse,
		"__native_json_stringify": nativeJsonStringify,
		"__native_json_pretty":    nativeJsonPretty,
		// Regex
		"__native_regex_match":   nativeRegexMatch,
		"__native_regex_find":    nativeRegexFind,
		"__native_regex_replace": nativeRegexReplace,
		// HTTP
		"__native_http_request": nativeHttpRequest,
	}

	for name, fn := range natives {
		builtins[name] = fn
	}
}

func expectNum(val object.Object, fn string) *object.NumberObj {
	n, ok := val.(*object.NumberObj)
	if !ok {
		runtimeError("%s() requires a num, got %s", fn, object.TypeName(val))
	}
	return n
}

func expectStr(val object.Object, fn string) *object.StringObj {
	s, ok := val.(*object.StringObj)
	if !ok {
		runtimeError("%s() requires a str, got %s", fn, object.TypeName(val))
	}
	return s
}

func nativeMath1(fn func(float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		val := expectNum(evalExpr(args[0], env), "math")
		return &object.NumberObj{Value: fn(val.Value)}
	}
}

func nativeMath2(fn func(float64, float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		a := expectNum(evalExpr(args[0], env), "math")
		b := expectNum(evalExpr(args[1], env), "math")
		return &object.NumberObj{Value: fn(a.Value, b.Value)}
	}
}

func nativeRandom(args []*parser.Expr, env *Environment) object.Object {
	return &object.NumberObj{Value: rand.Float64()}
}

func nativeUpper(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "upper")
	return &object.StringObj{Value: strings.ToUpper(s.Value)}
}

func nativeLower(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "lower")
	return &object.StringObj{Value: strings.ToLower(s.Value)}
}

func nativeSplit(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "split")
	sep := expectStr(evalExpr(args[1], env), "split")
	parts := strings.Split(s.Value, sep.Value)
	elements := make([]object.Object, len(parts))
	for i, p := range parts {
		elements[i] = &object.StringObj{Value: p}
	}
	return &object.ListObj{Elements: elements}
}

func nativeReplace(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "replace")
	old := expectStr(evalExpr(args[1], env), "replace")
	new_ := expectStr(evalExpr(args[2], env), "replace")
	return &object.StringObj{Value: strings.ReplaceAll(s.Value, old.Value, new_.Value)}
}

func nativeTrim(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "trim")
	return &object.StringObj{Value: strings.TrimSpace(s.Value)}
}

func nativeCharCode(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "char_code")
	if len(s.Value) == 0 {
		return &object.ErrorObj{Message: "char_code() called on empty string"}
	}
	runes := []rune(s.Value)
	return &object.NumberObj{Value: float64(runes[0])}
}

func nativeFromCharCode(args []*parser.Expr, env *Environment) object.Object {
	n := expectNum(evalExpr(args[0], env), "from_char_code")
	code := int(n.Value)
	if code < 0 || code > 0x10FFFF {
		return &object.ErrorObj{Message: fmt.Sprintf("from_char_code() code point %d out of range", code)}
	}
	return &object.StringObj{Value: string(rune(code))}
}

func nativeStrReverse(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "str_reverse")
	runes := []rune(s.Value)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return &object.StringObj{Value: string(runes)}
}

func nativeStrRepeat(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "str_repeat")
	n := expectNum(evalExpr(args[1], env), "str_repeat")
	count := int(n.Value)
	if count < 0 {
		return &object.ErrorObj{Message: "str_repeat() count must be >= 0"}
	}
	if count > 1<<20 && len(s.Value) > 0 {
		return &object.ErrorObj{Message: "str_repeat() result too large"}
	}
	return &object.StringObj{Value: strings.Repeat(s.Value, count)}
}

func nativeIndexOf(args []*parser.Expr, env *Environment) object.Object {
	haystack := evalExpr(args[0], env)
	needle := evalExpr(args[1], env)

	switch h := haystack.(type) {
	case *object.StringObj:
		n, ok := needle.(*object.StringObj)
		if !ok {
			runtimeError("index_of() searching in str requires a str needle, got %s", object.TypeName(needle))
		}
		idx := strings.Index(h.Value, n.Value)
		return &object.NumberObj{Value: float64(idx)}
	case *object.ListObj:
		for i, el := range h.Elements {
			if objectsEqual(el, needle) {
				return &object.NumberObj{Value: float64(i)}
			}
		}
		return &object.NumberObj{Value: -1}
	default:
		runtimeError("index_of() requires a str or list as first argument, got %s", object.TypeName(haystack))
	}
	return object.Null
}

func nativeSlice(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	startN := expectNum(evalExpr(args[1], env), "slice")
	endN := expectNum(evalExpr(args[2], env), "slice")

	switch v := val.(type) {
	case *object.StringObj:
		runes := []rune(v.Value)
		length := len(runes)
		start, end := clampSlice(int(startN.Value), int(endN.Value), length)
		return &object.StringObj{Value: string(runes[start:end])}
	case *object.ListObj:
		length := len(v.Elements)
		start, end := clampSlice(int(startN.Value), int(endN.Value), length)
		sliced := make([]object.Object, end-start)
		copy(sliced, v.Elements[start:end])
		return &object.ListObj{Elements: sliced}
	default:
		runtimeError("slice() requires a str or list as first argument, got %s", object.TypeName(val))
	}
	return object.Null
}

func clampSlice(start, end, length int) (int, int) {
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	if start > end {
		start = end
	}
	return start, end
}

func objectsEqual(a, b object.Object) bool {
	switch av := a.(type) {
	case *object.NumberObj:
		bv, ok := b.(*object.NumberObj)
		return ok && av.Value == bv.Value
	case *object.StringObj:
		bv, ok := b.(*object.StringObj)
		return ok && av.Value == bv.Value
	case *object.BoolObj:
		bv, ok := b.(*object.BoolObj)
		return ok && av.Value == bv.Value
	case *object.NullObj:
		_, ok := b.(*object.NullObj)
		return ok
	default:
		return a == b
	}
}

func nativeListConcat(args []*parser.Expr, env *Environment) object.Object {
	a := evalExpr(args[0], env)
	b := evalExpr(args[1], env)
	la, ok := a.(*object.ListObj)
	if !ok {
		runtimeError("list_concat() first argument must be a list, got %s", object.TypeName(a))
	}
	lb, ok := b.(*object.ListObj)
	if !ok {
		runtimeError("list_concat() second argument must be a list, got %s", object.TypeName(b))
	}
	result := make([]object.Object, 0, len(la.Elements)+len(lb.Elements))
	result = append(result, la.Elements...)
	result = append(result, lb.Elements...)
	return &object.ListObj{Elements: result}
}

func nativeListInsert(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	idxN := expectNum(evalExpr(args[1], env), "list_insert")
	item := evalExpr(args[2], env)

	l, ok := val.(*object.ListObj)
	if !ok {
		runtimeError("list_insert() first argument must be a list, got %s", object.TypeName(val))
	}
	idx := int(idxN.Value)
	length := len(l.Elements)
	if idx < 0 {
		idx = length + idx
	}
	if idx < 0 {
		idx = 0
	}
	if idx > length {
		idx = length
	}

	l.Elements = append(l.Elements, nil)
	copy(l.Elements[idx+1:], l.Elements[idx:])
	l.Elements[idx] = item
	return l
}

func nativeListReverse(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	l, ok := val.(*object.ListObj)
	if !ok {
		runtimeError("list_reverse() requires a list, got %s", object.TypeName(val))
	}
	for i, j := 0, len(l.Elements)-1; i < j; i, j = i+1, j-1 {
		l.Elements[i], l.Elements[j] = l.Elements[j], l.Elements[i]
	}
	return l
}

func nativeDictDelete(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	key := expectStr(evalExpr(args[1], env), "dict_delete")

	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("dict_delete() first argument must be a dict, got %s", object.TypeName(val))
	}
	removed, exists := d.Pairs[key.Value]
	if !exists {
		return object.Null
	}
	delete(d.Pairs, key.Value)
	for i, k := range d.Keys {
		if k == key.Value {
			d.Keys = append(d.Keys[:i], d.Keys[i+1:]...)
			break
		}
	}
	return removed
}

func nativeDictMerge(args []*parser.Expr, env *Environment) object.Object {
	a := evalExpr(args[0], env)
	b := evalExpr(args[1], env)
	da, ok := a.(*object.DictObj)
	if !ok {
		runtimeError("dict_merge() first argument must be a dict, got %s", object.TypeName(a))
	}
	db, ok := b.(*object.DictObj)
	if !ok {
		runtimeError("dict_merge() second argument must be a dict, got %s", object.TypeName(b))
	}
	for _, k := range db.Keys {
		if _, exists := da.Pairs[k]; !exists {
			da.Keys = append(da.Keys, k)
		}
		da.Pairs[k] = db.Pairs[k]
	}
	return da
}

func nativeReadFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "read")
	data, err := os.ReadFile(path.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Cannot read file: " + path.Value}
	}
	return &object.StringObj{Value: string(data)}
}

func nativeWriteFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "write")
	data := evalExpr(args[1], env)
	if err := os.WriteFile(path.Value, []byte(data.Inspect()), 0644); err != nil {
		return &object.ErrorObj{Message: "Cannot write file: " + path.Value}
	}
	return object.Null
}

func nativeFileExists(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "exists")
	_, err := os.Stat(path.Value)
	return &object.BoolObj{Value: err == nil}
}

func nativeDeleteFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "delete")
	if err := os.Remove(path.Value); err != nil {
		return &object.ErrorObj{Message: "Cannot delete file: " + path.Value}
	}
	return object.Null
}

func nativeCwd(args []*parser.Expr, env *Environment) object.Object {
	dir, _ := os.Getwd()
	return &object.StringObj{Value: dir}
}

func nativeGetenv(args []*parser.Expr, env *Environment) object.Object {
	name := expectStr(evalExpr(args[0], env), "getenv")
	val := os.Getenv(name.Value)
	if val == "" {
		return object.Null
	}
	return &object.StringObj{Value: val}
}

func nativeExec(args []*parser.Expr, env *Environment) object.Object {
	cmdStr := expectStr(evalExpr(args[0], env), "exec")
	parts := strings.Fields(cmdStr.Value)
	if len(parts) == 0 {
		return &object.StringObj{Value: ""}
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	output, _ := cmd.CombinedOutput()
	return &object.StringObj{Value: strings.TrimRight(string(output), "\n")}
}

func nativeTime(args []*parser.Expr, env *Environment) object.Object {
	return &object.NumberObj{Value: float64(time.Now().Unix())}
}

func nativeSleep(args []*parser.Expr, env *Environment) object.Object {
	ms := expectNum(evalExpr(args[0], env), "sleep")
	if ms.Value < 0 {
		return &object.ErrorObj{Message: "sleep() duration must be >= 0"}
	}
	if ms.Value > 86400000 {
		return &object.ErrorObj{Message: "sleep() duration too large (max 86400000ms)"}
	}
	time.Sleep(time.Duration(ms.Value) * time.Millisecond)
	return object.Null
}

func nativeExit(args []*parser.Expr, env *Environment) object.Object {
	code := expectNum(evalExpr(args[0], env), "exit")
	os.Exit(int(code.Value))
	return object.Null
}

func nativeFormatTime(args []*parser.Expr, env *Environment) object.Object {
	ts := expectNum(evalExpr(args[0], env), "format_time")
	fmtStr := expectStr(evalExpr(args[1], env), "format_time")
	t := time.Unix(int64(ts.Value), 0)
	layout := fmtStr.Value
	layout = strings.ReplaceAll(layout, "YYYY", "2006")
	layout = strings.ReplaceAll(layout, "MM", "01")
	layout = strings.ReplaceAll(layout, "DD", "02")
	layout = strings.ReplaceAll(layout, "hh", "15")
	layout = strings.ReplaceAll(layout, "mm", "04")
	layout = strings.ReplaceAll(layout, "ss", "05")
	layout = strings.ReplaceAll(layout, "tz", "MST")

	return &object.StringObj{Value: t.Format(layout)}
}

func nativeRename(args []*parser.Expr, env *Environment) object.Object {
	oldPath := expectStr(evalExpr(args[0], env), "rename")
	newPath := expectStr(evalExpr(args[1], env), "rename")
	if err := os.Rename(oldPath.Value, newPath.Value); err != nil {
		return &object.ErrorObj{Message: "Cannot rename: " + err.Error()}
	}
	return object.Null
}

func nativeJsonParse(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "json/parse")
	dec := json.NewDecoder(strings.NewReader(s.Value))
	dec.UseNumber()
	result, err := jsonDecodeValue(dec)
	if err != nil {
		return &object.ErrorObj{Message: "Invalid JSON: " + err.Error()}
	}
	return result
}

func jsonDecodeValue(dec *json.Decoder) (object.Object, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			return jsonDecodeObject(dec)
		case '[':
			return jsonDecodeArray(dec)
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

func jsonDecodeObject(dec *json.Decoder) (object.Object, error) {
	pairs := make(map[string]object.Object)
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key := tok.(string)
		keys = append(keys, key)
		val, err := jsonDecodeValue(dec)
		if err != nil {
			return nil, err
		}
		pairs[key] = val
	}
	dec.Token()
	return &object.DictObj{Pairs: pairs, Keys: keys}, nil
}

func jsonDecodeArray(dec *json.Decoder) (object.Object, error) {
	var elements []object.Object
	for dec.More() {
		val, err := jsonDecodeValue(dec)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}
	dec.Token()
	return &object.ListObj{Elements: elements}, nil
}

func nativeJsonStringify(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: jsonMarshal(val, "", "")}
}

func nativeJsonPretty(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: jsonMarshal(val, "", "  ")}
}

func jsonMarshal(obj object.Object, prefix, indent string) string {
	var buf strings.Builder
	jsonWriteValue(&buf, obj, prefix, indent, 0)
	return buf.String()
}

func jsonWriteValue(buf *strings.Builder, obj object.Object, prefix, indent string, depth int) {
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
			jsonWriteValue(buf, el, prefix, indent, depth+1)
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
			jsonWriteValue(buf, v.Pairs[k], prefix, indent, depth+1)
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

func nativeRegexMatch(args []*parser.Expr, env *Environment) object.Object {
	pattern := expectStr(evalExpr(args[0], env), "regex_match")
	str := expectStr(evalExpr(args[1], env), "regex_match")
	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Invalid regex: " + err.Error()}
	}
	return &object.BoolObj{Value: re.MatchString(str.Value)}
}

func nativeRegexFind(args []*parser.Expr, env *Environment) object.Object {
	pattern := expectStr(evalExpr(args[0], env), "regex_find")
	str := expectStr(evalExpr(args[1], env), "regex_find")
	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Invalid regex: " + err.Error()}
	}
	matches := re.FindAllString(str.Value, -1)
	elements := make([]object.Object, len(matches))
	for i, m := range matches {
		elements[i] = &object.StringObj{Value: m}
	}
	return &object.ListObj{Elements: elements}
}

func nativeRegexReplace(args []*parser.Expr, env *Environment) object.Object {
	pattern := expectStr(evalExpr(args[0], env), "regex_replace")
	str := expectStr(evalExpr(args[1], env), "regex_replace")
	replacement := expectStr(evalExpr(args[2], env), "regex_replace")
	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Invalid regex: " + err.Error()}
	}
	return &object.StringObj{Value: re.ReplaceAllString(str.Value, replacement.Value)}
}

func nativeHttpRequest(args []*parser.Expr, env *Environment) object.Object {
	method := expectStr(evalExpr(args[0], env), "http_request")
	url := expectStr(evalExpr(args[1], env), "http_request")
	headersVal := evalExpr(args[2], env)
	bodyVal := evalExpr(args[3], env)

	var bodyReader io.Reader
	if bodyStr, ok := bodyVal.(*object.StringObj); ok && bodyStr.Value != "" {
		bodyReader = strings.NewReader(bodyStr.Value)
	}

	req, err := http.NewRequest(method.Value, url.Value, bodyReader)
	if err != nil {
		return &object.ErrorObj{Message: "HTTP error: " + err.Error()}
	}

	if headers, ok := headersVal.(*object.DictObj); ok {
		for _, k := range headers.Keys {
			if v, ok := headers.Pairs[k].(*object.StringObj); ok {
				req.Header.Set(k, v.Value)
			}
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &object.ErrorObj{Message: "HTTP error: " + err.Error()}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &object.ErrorObj{Message: "HTTP read error: " + err.Error()}
	}
	return &object.StringObj{Value: string(data)}
}
