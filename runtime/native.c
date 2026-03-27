/*
 * Native bridge functions (__native_*).
 * These are thin C wrappers around system libraries.
 * The Sintax stdlib (.sx files) calls these via the interpreter,
 * and the codegen maps them directly for compiled binaries.
 */

#include "runtime.h"
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <ctype.h>
#include <sys/stat.h>
#include <unistd.h>
#include <time.h>
#include <regex.h>

#ifdef SX_USE_CURL
#include <curl/curl.h>
#endif

/* --- Type check helpers --- */

static void expect_num(SxValue *v, const char *fn) {
    if (!v || v->type != SX_NUMBER) {
        char msg[128];
        snprintf(msg, sizeof(msg), "%s() requires a num, got %s",
                 fn, v ? sx_type(v)->string : "null");
        sx_error(msg);
    }
}

static void expect_str(SxValue *v, const char *fn) {
    if (!v || v->type != SX_STRING) {
        char msg[128];
        snprintf(msg, sizeof(msg), "%s() requires a str, got %s",
                 fn, v ? sx_type(v)->string : "null");
        sx_error(msg);
    }
}

/* --- Math natives --- */

SxValue* __native_sqrt(SxValue *v) {
    expect_num(v, "sqrt");
    return sx_number(sqrt(v->number));
}

SxValue* __native_sin(SxValue *v) {
    expect_num(v, "sin");
    return sx_number(sin(v->number));
}

SxValue* __native_cos(SxValue *v) {
    expect_num(v, "cos");
    return sx_number(cos(v->number));
}

SxValue* __native_tan(SxValue *v) {
    expect_num(v, "tan");
    return sx_number(tan(v->number));
}

SxValue* __native_asin(SxValue *v) {
    expect_num(v, "asin");
    return sx_number(asin(v->number));
}

SxValue* __native_acos(SxValue *v) {
    expect_num(v, "acos");
    return sx_number(acos(v->number));
}

SxValue* __native_atan(SxValue *v) {
    expect_num(v, "atan");
    return sx_number(atan(v->number));
}

SxValue* __native_log(SxValue *v) {
    expect_num(v, "log");
    return sx_number(log(v->number));
}

SxValue* __native_log2(SxValue *v) {
    expect_num(v, "log2");
    return sx_number(log2(v->number));
}

SxValue* __native_log10(SxValue *v) {
    expect_num(v, "log10");
    return sx_number(log10(v->number));
}

SxValue* __native_exp(SxValue *v) {
    expect_num(v, "exp");
    return sx_number(exp(v->number));
}

SxValue* __native_floor(SxValue *v) {
    expect_num(v, "floor");
    return sx_number(floor(v->number));
}

SxValue* __native_ceil(SxValue *v) {
    expect_num(v, "ceil");
    return sx_number(ceil(v->number));
}

SxValue* __native_round(SxValue *v) {
    expect_num(v, "round");
    return sx_number(round(v->number));
}

SxValue* __native_cbrt(SxValue *v) {
    expect_num(v, "cbrt");
    return sx_number(cbrt(v->number));
}

SxValue* __native_pow(SxValue *a, SxValue *b) {
    expect_num(a, "pow");
    expect_num(b, "pow");
    return sx_number(pow(a->number, b->number));
}

static int _random_seeded = 0;
SxValue* __native_random(void) {
    if (!_random_seeded) {
        srand((unsigned int)time(NULL));
        _random_seeded = 1;
    }
    return sx_number((double)rand() / RAND_MAX);
}

/* --- String natives --- */

SxValue* __native_upper(SxValue *v) {
    expect_str(v, "upper");
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = toupper((unsigned char)s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* __native_lower(SxValue *v) {
    expect_str(v, "lower");
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = tolower((unsigned char)s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* __native_split(SxValue *v, SxValue *sep) {
    expect_str(v, "split");
    expect_str(sep, "split");
    SxValue *list = sx_list_new();
    char *s = sx_strdup(v->string);
    char *delim = sep->string;
    int dlen = strlen(delim);
    char *start = s;
    char *found;
    while ((found = strstr(start, delim)) != NULL) {
        *found = '\0';
        sx_list_append(list, sx_string(start));
        start = found + dlen;
    }
    sx_list_append(list, sx_string(start));
    SX_FREE(s);
    return list;
}

SxValue* __native_replace(SxValue *v, SxValue *old, SxValue *new_) {
    expect_str(v, "replace");
    expect_str(old, "replace");
    expect_str(new_, "replace");
    char *s = v->string, *o = old->string, *n = new_->string;
    size_t olen = strlen(o), nlen = strlen(n), slen = strlen(s);
    if (olen == 0) return sx_string(s);

    size_t count = 0;
    char *p = s;
    while ((p = strstr(p, o)) != NULL) { count++; p += olen; }

    size_t result_len = slen + count * (nlen > olen ? nlen - olen : 0)
                             - count * (olen > nlen ? olen - nlen : 0);
    char *result = (char*)SX_MALLOC(result_len + 1);
    if (!result) { fprintf(stderr, "Error: out of memory\n"); exit(1); }
    char *dst = result;
    p = s;
    char *found;
    while ((found = strstr(p, o)) != NULL) {
        size_t chunk = found - p;
        memcpy(dst, p, chunk); dst += chunk;
        memcpy(dst, n, nlen); dst += nlen;
        p = found + olen;
    }
    strcpy(dst, p);
    SxValue *r = sx_alloc(SX_STRING); r->string = result; return r;
}

/* --- OS natives --- */

SxValue* __native_read_file(SxValue *path) {
    expect_str(path, "read");
    FILE *f = fopen(path->string, "r");
    if (!f) return sx_error_new(sx_string("Cannot read file"));

    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    if (len < 0) {
        fclose(f);
        f = fopen(path->string, "r");
        if (!f) return sx_error_new(sx_string("Cannot read file"));
        size_t cap = 4096, total = 0;
        char *buf = (char*)SX_MALLOC(cap);
        if (!buf) { fclose(f); return sx_error_new(sx_string("out of memory")); }
        size_t n;
        while ((n = fread(buf + total, 1, cap - total, f)) > 0) {
            total += n;
            if (total >= cap) {
                cap *= 2;
                char *tmp = (char*)SX_REALLOC(buf, cap);
                if (!tmp) { SX_FREE(buf); fclose(f); return sx_error_new(sx_string("out of memory")); }
                buf = tmp;
            }
        }
        // Ensure room for null terminator
        if (total >= cap) {
            buf = (char*)SX_REALLOC(buf, total + 1);
        }
        buf[total] = '\0';
        fclose(f);
        SxValue *v = sx_alloc(SX_STRING); v->string = buf; return v;
    }

    fseek(f, 0, SEEK_SET);
    char *buf = (char*)SX_MALLOC(len + 1);
    fread(buf, 1, len, f);
    buf[len] = '\0';
    fclose(f);
    SxValue *v = sx_alloc(SX_STRING); v->string = buf; return v;
}

SxValue* __native_write_file(SxValue *path, SxValue *data) {
    expect_str(path, "write");
    FILE *f = fopen(path->string, "w");
    if (!f) return sx_error_new(sx_string("Cannot write file"));
    SxValue *s = sx_to_string(data);
    fputs(s->string, f);
    fclose(f);
    return sx_null();
}

SxValue* __native_file_exists(SxValue *path) {
    expect_str(path, "exists");
    struct stat st;
    return sx_bool(stat(path->string, &st) == 0);
}

SxValue* __native_delete_file(SxValue *path) {
    expect_str(path, "delete");
    if (remove(path->string) != 0) return sx_error_new(sx_string("Cannot delete file"));
    return sx_null();
}

SxValue* __native_cwd(void) {
    char buf[4096];
    if (getcwd(buf, sizeof(buf))) return sx_string(buf);
    return sx_string(".");
}

SxValue* __native_getenv(SxValue *name) {
    expect_str(name, "getenv");
    char *val = getenv(name->string);
    if (!val) return sx_null();
    return sx_string(val);
}

SxValue* __native_exec(SxValue *cmd) {
    expect_str(cmd, "exec");
    FILE *f = popen(cmd->string, "r");
    if (!f) return sx_string("");
    size_t cap = 4096, total = 0;
    char *result = (char*)SX_MALLOC(cap);
    char buf[4096];
    while (fgets(buf, sizeof(buf), f)) {
        size_t len = strlen(buf);
        if (total + len >= cap) {
            cap = (total + len) * 2;
            char *new_result = (char*)SX_REALLOC(result, cap);
            if (!new_result) { SX_FREE(result); pclose(f); return sx_string(""); }
            result = new_result;
        }
        memcpy(result + total, buf, len);
        total += len;
    }
    result[total] = '\0';
    if (total > 0 && result[total-1] == '\n') result[total-1] = '\0';
    pclose(f);
    SxValue *v = sx_alloc(SX_STRING); v->string = result; return v;
}

SxValue* __native_time(void) {
    return sx_number((double)time(NULL));
}

/* --- JSON natives --- */

static SxValue* json_parse_value(const char **p);
static void json_stringify_value(SxValue *v, char **buf, int *len, int *cap, const char *indent, int depth);

static void json_skip_ws(const char **p) {
    while (**p == ' ' || **p == '\t' || **p == '\n' || **p == '\r') (*p)++;
}

static void buf_append(char **buf, int *len, int *cap, const char *s, int slen) {
    while (*len + slen >= *cap) {
        *cap = (*cap == 0) ? 256 : *cap * 2;
        *buf = (char*)SX_REALLOC(*buf, *cap);
    }
    memcpy(*buf + *len, s, slen);
    *len += slen;
}

static void buf_append_str(char **buf, int *len, int *cap, const char *s) {
    buf_append(buf, len, cap, s, strlen(s));
}

static void buf_append_char(char **buf, int *len, int *cap, char c) {
    buf_append(buf, len, cap, &c, 1);
}

static SxValue* json_parse_string(const char **p) {
    (*p)++;
    int cap = 64, len = 0;
    char *buf = (char*)SX_MALLOC(cap);
    while (**p && **p != '"') {
        if (**p == '\\') {
            (*p)++;
            switch (**p) {
                case '"': buf_append_char(&buf, &len, &cap, '"'); break;
                case '\\': buf_append_char(&buf, &len, &cap, '\\'); break;
                case '/': buf_append_char(&buf, &len, &cap, '/'); break;
                case 'n': buf_append_char(&buf, &len, &cap, '\n'); break;
                case 't': buf_append_char(&buf, &len, &cap, '\t'); break;
                case 'r': buf_append_char(&buf, &len, &cap, '\r'); break;
                case 'b': buf_append_char(&buf, &len, &cap, '\b'); break;
                case 'f': buf_append_char(&buf, &len, &cap, '\f'); break;
                default: buf_append_char(&buf, &len, &cap, **p); break;
            }
        } else {
            buf_append_char(&buf, &len, &cap, **p);
        }
        (*p)++;
    }
    if (**p == '"') (*p)++;
    buf[len] = '\0';
    SxValue *v = sx_alloc(SX_STRING);
    v->string = buf;
    return v;
}

static SxValue* json_parse_number(const char **p) {
    char *end;
    double n = strtod(*p, &end);
    *p = end;
    return sx_number(n);
}

static SxValue* json_parse_array(const char **p) {
    (*p)++;
    SxValue *list = sx_list_new();
    json_skip_ws(p);
    if (**p == ']') { (*p)++; return list; }
    while (1) {
        json_skip_ws(p);
        SxValue *val = json_parse_value(p);
        sx_list_append(list, val);
        json_skip_ws(p);
        if (**p == ',') { (*p)++; continue; }
        if (**p == ']') { (*p)++; break; }
        break;
    }
    return list;
}

static SxValue* json_parse_object(const char **p) {
    (*p)++;
    SxValue *dict = sx_dict_new();
    json_skip_ws(p);
    if (**p == '}') { (*p)++; return dict; }
    while (1) {
        json_skip_ws(p);
        if (**p != '"') break;
        SxValue *key = json_parse_string(p);
        json_skip_ws(p);
        if (**p == ':') (*p)++;
        json_skip_ws(p);
        SxValue *val = json_parse_value(p);
        sx_dict_set(dict, key, val);
        json_skip_ws(p);
        if (**p == ',') { (*p)++; continue; }
        if (**p == '}') { (*p)++; break; }
        break;
    }
    return dict;
}

static SxValue* json_parse_value(const char **p) {
    json_skip_ws(p);
    switch (**p) {
        case '"': return json_parse_string(p);
        case '{': return json_parse_object(p);
        case '[': return json_parse_array(p);
        case 't':
            if (strncmp(*p, "true", 4) == 0) { *p += 4; return sx_bool(1); }
            return sx_null();
        case 'f':
            if (strncmp(*p, "false", 5) == 0) { *p += 5; return sx_bool(0); }
            return sx_null();
        case 'n':
            if (strncmp(*p, "null", 4) == 0) { *p += 4; return sx_null(); }
            return sx_null();
        default:
            if (**p == '-' || (**p >= '0' && **p <= '9'))
                return json_parse_number(p);
            return sx_null();
    }
}

SxValue* __native_json_parse(SxValue *s) {
    expect_str(s, "json/parse");
    const char *p = s->string;
    return json_parse_value(&p);
}

static void json_escape_string(const char *s, char **buf, int *len, int *cap) {
    buf_append_char(buf, len, cap, '"');
    for (int i = 0; s[i]; i++) {
        switch (s[i]) {
            case '"':  buf_append_str(buf, len, cap, "\\\""); break;
            case '\\': buf_append_str(buf, len, cap, "\\\\"); break;
            case '\n': buf_append_str(buf, len, cap, "\\n"); break;
            case '\t': buf_append_str(buf, len, cap, "\\t"); break;
            case '\r': buf_append_str(buf, len, cap, "\\r"); break;
            case '\b': buf_append_str(buf, len, cap, "\\b"); break;
            case '\f': buf_append_str(buf, len, cap, "\\f"); break;
            default:   buf_append_char(buf, len, cap, s[i]); break;
        }
    }
    buf_append_char(buf, len, cap, '"');
}

static void json_add_indent(char **buf, int *len, int *cap, const char *indent, int depth) {
    if (!indent) return;
    buf_append_char(buf, len, cap, '\n');
    for (int i = 0; i < depth; i++)
        buf_append_str(buf, len, cap, indent);
}

static void json_stringify_value(SxValue *v, char **buf, int *len, int *cap, const char *indent, int depth) {
    if (!v) { buf_append_str(buf, len, cap, "null"); return; }
    switch (v->type) {
        case SX_NULL:
        case SX_ERROR:
            buf_append_str(buf, len, cap, "null");
            break;
        case SX_BOOL:
            buf_append_str(buf, len, cap, v->boolean ? "true" : "false");
            break;
        case SX_NUMBER: {
            char tmp[64];
            if (v->number == (long long)v->number)
                snprintf(tmp, sizeof(tmp), "%lld", (long long)v->number);
            else
                snprintf(tmp, sizeof(tmp), "%g", v->number);
            buf_append_str(buf, len, cap, tmp);
            break;
        }
        case SX_STRING:
            json_escape_string(v->string, buf, len, cap);
            break;
        case SX_LIST:
            buf_append_char(buf, len, cap, '[');
            for (int i = 0; i < v->list.len; i++) {
                if (i > 0) buf_append_char(buf, len, cap, ',');
                if (indent) json_add_indent(buf, len, cap, indent, depth + 1);
                json_stringify_value(v->list.items[i], buf, len, cap, indent, depth + 1);
            }
            if (v->list.len > 0 && indent) json_add_indent(buf, len, cap, indent, depth);
            buf_append_char(buf, len, cap, ']');
            break;
        case SX_DICT:
            buf_append_char(buf, len, cap, '{');
            for (int i = 0; i < v->dict.len; i++) {
                if (i > 0) buf_append_char(buf, len, cap, ',');
                if (indent) json_add_indent(buf, len, cap, indent, depth + 1);
                json_escape_string(v->dict.keys[i], buf, len, cap);
                buf_append_char(buf, len, cap, ':');
                if (indent) buf_append_char(buf, len, cap, ' ');
                SxDictBucket *b = sx_dict_find(&v->dict, v->dict.keys[i]);
                json_stringify_value(b ? b->value : sx_null(), buf, len, cap, indent, depth + 1);
            }
            if (v->dict.len > 0 && indent) json_add_indent(buf, len, cap, indent, depth);
            buf_append_char(buf, len, cap, '}');
            break;
        default:
            buf_append_str(buf, len, cap, "null");
            break;
    }
}

SxValue* __native_json_stringify(SxValue *v) {
    char *buf = NULL;
    int len = 0, cap = 0;
    json_stringify_value(v, &buf, &len, &cap, NULL, 0);
    buf_append_char(&buf, &len, &cap, '\0');
    SxValue *r = sx_alloc(SX_STRING);
    r->string = buf;
    return r;
}

SxValue* __native_json_pretty(SxValue *v) {
    char *buf = NULL;
    int len = 0, cap = 0;
    json_stringify_value(v, &buf, &len, &cap, "  ", 0);
    buf_append_char(&buf, &len, &cap, '\0');
    SxValue *r = sx_alloc(SX_STRING);
    r->string = buf;
    return r;
}

/* --- String building block natives --- */

SxValue* __native_trim(SxValue *v) {
    expect_str(v, "trim");
    const char *s = v->string;
    int len = strlen(s);
    int start = 0, end = len - 1;
    while (start < len && isspace((unsigned char)s[start])) start++;
    while (end > start && isspace((unsigned char)s[end])) end--;
    int newlen = end - start + 1;
    if (newlen <= 0) return sx_string("");
    char *buf = (char*)SX_MALLOC(newlen + 1);
    memcpy(buf, s + start, newlen);
    buf[newlen] = '\0';
    SxValue *r = sx_alloc(SX_STRING); r->string = buf; return r;
}

SxValue* __native_char_code(SxValue *v) {
    expect_str(v, "char_code");
    if (v->string[0] == '\0')
        return sx_error_new(sx_string("char_code() called on empty string"));
    return sx_number((double)(unsigned char)v->string[0]);
}

SxValue* __native_from_char_code(SxValue *v) {
    expect_num(v, "from_char_code");
    int code = (int)v->number;
    if (code < 0 || code > 255) {
        return sx_error_new(sx_string("from_char_code: code out of range (0-255)"));
    }
    char buf[2] = {(char)code, '\0'};
    return sx_string(buf);
}

SxValue* __native_str_reverse(SxValue *v) {
    expect_str(v, "str_reverse");
    int len = strlen(v->string);
    char *buf = (char*)SX_MALLOC(len + 1);
    for (int i = 0; i < len; i++)
        buf[i] = v->string[len - 1 - i];
    buf[len] = '\0';
    SxValue *r = sx_alloc(SX_STRING); r->string = buf; return r;
}

SxValue* __native_str_repeat(SxValue *v, SxValue *n) {
    expect_str(v, "str_repeat");
    expect_num(n, "str_repeat");
    int count = (int)n->number;
    if (count < 0) return sx_error_new(sx_string("str_repeat() count must be >= 0"));
    size_t slen = strlen(v->string);
    size_t total = slen * (size_t)count;
    if (count > 0 && total / (size_t)count != slen) return sx_error_new(sx_string("str_repeat: result too large"));
    if (total > 100 * 1024 * 1024) return sx_error_new(sx_string("str_repeat: result too large"));
    char *buf = (char*)SX_MALLOC(total + 1);
    if (!buf) { fprintf(stderr, "Error: out of memory\n"); exit(1); }
    for (int i = 0; i < count; i++)
        memcpy(buf + (size_t)i * slen, v->string, slen);
    buf[total] = '\0';
    SxValue *r = sx_alloc(SX_STRING); r->string = buf; return r;
}

SxValue* __native_index_of(SxValue *haystack, SxValue *needle) {
    if (haystack->type == SX_STRING) {
        expect_str(needle, "index_of");
        const char *found = strstr(haystack->string, needle->string);
        if (!found) return sx_number(-1);
        return sx_number((double)(found - haystack->string));
    }
    if (haystack->type == SX_LIST) {
        for (int i = 0; i < haystack->list.len; i++) {
            SxValue *eq = sx_eq(haystack->list.items[i], needle);
            if (eq->boolean) return sx_number((double)i);
        }
        return sx_number(-1);
    }
    sx_error("index_of() requires a str or list");
    return sx_null();
}

static int clamp_idx(int idx, int len) {
    if (idx < 0) idx = len + idx;
    if (idx < 0) idx = 0;
    if (idx > len) idx = len;
    return idx;
}

SxValue* __native_slice(SxValue *v, SxValue *start_v, SxValue *end_v) {
    expect_num(start_v, "slice");
    expect_num(end_v, "slice");
    int s = (int)start_v->number;
    int e = (int)end_v->number;

    if (v->type == SX_STRING) {
        int len = strlen(v->string);
        s = clamp_idx(s, len);
        e = clamp_idx(e, len);
        if (s >= e) return sx_string("");
        int newlen = e - s;
        char *buf = (char*)SX_MALLOC(newlen + 1);
        memcpy(buf, v->string + s, newlen);
        buf[newlen] = '\0';
        SxValue *r = sx_alloc(SX_STRING); r->string = buf; return r;
    }
    if (v->type == SX_LIST) {
        int len = v->list.len;
        s = clamp_idx(s, len);
        e = clamp_idx(e, len);
        SxValue *list = sx_list_new();
        for (int i = s; i < e; i++)
            sx_list_append(list, v->list.items[i]);
        return list;
    }
    sx_error("slice() requires a str or list");
    return sx_null();
}

/* --- List building block natives --- */

SxValue* __native_list_concat(SxValue *a, SxValue *b) {
    if (!a || a->type != SX_LIST) sx_error("list_concat() first arg must be a list");
    if (!b || b->type != SX_LIST) sx_error("list_concat() second arg must be a list");
    SxValue *list = sx_list_new();
    for (int i = 0; i < a->list.len; i++)
        sx_list_append(list, a->list.items[i]);
    for (int i = 0; i < b->list.len; i++)
        sx_list_append(list, b->list.items[i]);
    return list;
}

SxValue* __native_list_insert(SxValue *list, SxValue *idx_v, SxValue *item) {
    if (!list || list->type != SX_LIST) sx_error("list_insert() first arg must be a list");
    expect_num(idx_v, "list_insert");
    int idx = (int)idx_v->number;
    int len = list->list.len;
    if (idx < 0) idx = len + idx;
    if (idx < 0) idx = 0;
    if (idx > len) idx = len;
    // Grow
    if (list->list.len >= list->list.cap) {
        list->list.cap = list->list.cap ? list->list.cap * 2 : 4;
        list->list.items = (SxValue**)SX_REALLOC(list->list.items,
            sizeof(SxValue*) * list->list.cap);
    }
    // Shift
    for (int i = list->list.len; i > idx; i--)
        list->list.items[i] = list->list.items[i - 1];
    list->list.items[idx] = item;
    list->list.len++;
    return list;
}

SxValue* __native_list_reverse(SxValue *list) {
    if (!list || list->type != SX_LIST) sx_error("list_reverse() requires a list");
    int len = list->list.len;
    for (int i = 0, j = len - 1; i < j; i++, j--) {
        SxValue *tmp = list->list.items[i];
        list->list.items[i] = list->list.items[j];
        list->list.items[j] = tmp;
    }
    return list;
}

/* --- Dict building block natives --- */

SxValue* __native_dict_delete(SxValue *dict, SxValue *key) {
    if (!dict || dict->type != SX_DICT) sx_error("dict_delete() first arg must be a dict");
    expect_str(key, "dict_delete");
    SxDictBucket *b = sx_dict_find(&dict->dict, key->string);
    if (!b || !b->used) return sx_null();
    SxValue *removed = b->value;
    // Mark bucket as tombstone so probing continues past it
    SX_FREE(b->key);
    b->key = NULL;
    b->value = NULL;
    b->used = 2; // tombstone
    // Remove from insertion-order keys array
    for (int i = 0; i < dict->dict.len; i++) {
        if (strcmp(dict->dict.keys[i], key->string) == 0) {
            SX_FREE(dict->dict.keys[i]);
            for (int j = i; j < dict->dict.len - 1; j++)
                dict->dict.keys[j] = dict->dict.keys[j + 1];
            dict->dict.len--;
            break;
        }
    }
    return removed;
}

SxValue* __native_dict_merge(SxValue *a, SxValue *b) {
    if (!a || a->type != SX_DICT) sx_error("dict_merge() first arg must be a dict");
    if (!b || b->type != SX_DICT) sx_error("dict_merge() second arg must be a dict");
    for (int i = 0; i < b->dict.len; i++) {
        SxDictBucket *bkt = sx_dict_find(&b->dict, b->dict.keys[i]);
        if (bkt) {
            SxValue *k = sx_string(b->dict.keys[i]);
            sx_dict_set(a, k, bkt->value);
        }
    }
    return a;
}

/* --- System building block natives --- */

SxValue* __native_sleep(SxValue *ms) {
    expect_num(ms, "sleep");
    if (ms->number < 0) return sx_error_new(sx_string("sleep() duration must be >= 0"));
    struct timespec ts;
    ts.tv_sec = (time_t)(ms->number / 1000);
    ts.tv_nsec = (long)((long long)(ms->number) % 1000 * 1000000);
    nanosleep(&ts, NULL);
    return sx_null();
}

SxValue* __native_exit(SxValue *code) {
    expect_num(code, "exit");
    exit((int)code->number);
    return sx_null();
}

SxValue* __native_format_time(SxValue *ts, SxValue *fmt) {
    expect_num(ts, "format_time");
    expect_str(fmt, "format_time");
    time_t t = (time_t)ts->number;
    struct tm *tm = localtime(&t);
    char buf[256];
    // Map Sintax tokens to strftime
    // Build a strftime format string from the Sintax format
    const char *src = fmt->string;
    char sfmt[256];
    int j = 0;
    for (int i = 0; src[i] && j < 250; ) {
        if (strncmp(src + i, "YYYY", 4) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'Y'; i += 4; }
        else if (strncmp(src + i, "MM", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'm'; i += 2; }
        else if (strncmp(src + i, "DD", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'd'; i += 2; }
        else if (strncmp(src + i, "hh", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'H'; i += 2; }
        else if (strncmp(src + i, "mm", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'M'; i += 2; }
        else if (strncmp(src + i, "ss", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'S'; i += 2; }
        else if (strncmp(src + i, "tz", 2) == 0) { sfmt[j++] = '%'; sfmt[j++] = 'Z'; i += 2; }
        else sfmt[j++] = src[i++];
    }
    sfmt[j] = '\0';
    if (strftime(buf, sizeof(buf), sfmt, tm) == 0) {
        buf[0] = '\0'; // Truncated — return empty rather than garbage
    }
    return sx_string(buf);
}

SxValue* __native_rename(SxValue *old, SxValue *new_) {
    expect_str(old, "rename");
    expect_str(new_, "rename");
    if (rename(old->string, new_->string) != 0)
        return sx_error_new(sx_string("Cannot rename file"));
    return sx_null();
}

/* --- Regex natives (POSIX regex.h) --- */

SxValue* __native_regex_match(SxValue *pattern, SxValue *str) {
    expect_str(pattern, "regex_match");
    expect_str(str, "regex_match");

    regex_t reg;
    int rc = regcomp(&reg, pattern->string, REG_EXTENDED | REG_NOSUB);
    if (rc != 0) {
        char errbuf[128];
        regerror(rc, &reg, errbuf, sizeof(errbuf));
        regfree(&reg);
        return sx_error_new(sx_string(errbuf));
    }

    int match = regexec(&reg, str->string, 0, NULL, 0);
    regfree(&reg);
    return sx_bool(match == 0);
}

SxValue* __native_regex_find(SxValue *pattern, SxValue *str) {
    expect_str(pattern, "regex_find");
    expect_str(str, "regex_find");

    regex_t reg;
    int rc = regcomp(&reg, pattern->string, REG_EXTENDED);
    if (rc != 0) {
        char errbuf[128];
        regerror(rc, &reg, errbuf, sizeof(errbuf));
        regfree(&reg);
        return sx_error_new(sx_string(errbuf));
    }

    SxValue *list = sx_list_new();
    const char *cursor = str->string;
    regmatch_t match;

    while (regexec(&reg, cursor, 1, &match, 0) == 0) {
        if (match.rm_so == match.rm_eo) {
            if (*cursor == '\0') break;
            cursor++;
            continue;
        }
        int len = match.rm_eo - match.rm_so;
        char *sub = (char *)SX_MALLOC(len + 1);
        memcpy(sub, cursor + match.rm_so, len);
        sub[len] = '\0';
        SxValue *sv = sx_alloc(SX_STRING);
        sv->string = sub;
        sx_list_append(list, sv);
        cursor += match.rm_eo;
    }

    regfree(&reg);
    return list;
}

SxValue* __native_regex_replace(SxValue *pattern, SxValue *str, SxValue *replacement) {
    expect_str(pattern, "regex_replace");
    expect_str(str, "regex_replace");
    expect_str(replacement, "regex_replace");

    regex_t reg;
    int rc = regcomp(&reg, pattern->string, REG_EXTENDED);
    if (rc != 0) {
        char errbuf[128];
        regerror(rc, &reg, errbuf, sizeof(errbuf));
        regfree(&reg);
        return sx_error_new(sx_string(errbuf));
    }

    const char *cursor = str->string;
    regmatch_t match;
    char *result = NULL;
    int rlen = 0, rcap = 0;

    while (regexec(&reg, cursor, 1, &match, 0) == 0) {
        if (match.rm_so > 0)
            buf_append(&result, &rlen, &rcap, cursor, match.rm_so);
        buf_append_str(&result, &rlen, &rcap, replacement->string);
        if (match.rm_so == match.rm_eo) {
            if (*cursor == '\0') break;
            buf_append(&result, &rlen, &rcap, cursor, 1);
            cursor++;
            continue;
        }
        cursor += match.rm_eo;
    }
    buf_append_str(&result, &rlen, &rcap, cursor);
    buf_append_char(&result, &rlen, &rcap, '\0');

    regfree(&reg);

    SxValue *r = sx_alloc(SX_STRING);
    r->string = result ? result : sx_strdup("");
    return r;
}

/* --- HTTP native (libcurl) --- */

#ifdef SX_USE_CURL

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} CurlBuffer;

static size_t curl_write_cb(void *ptr, size_t size, size_t nmemb, void *userdata) {
    CurlBuffer *buf = (CurlBuffer *)userdata;
    size_t total = size * nmemb;
    while (buf->len + total >= buf->cap) {
        buf->cap = buf->cap ? buf->cap * 2 : 4096;
        buf->data = (char *)SX_REALLOC(buf->data, buf->cap);
    }
    memcpy(buf->data + buf->len, ptr, total);
    buf->len += total;
    return total;
}

static int _curl_initialized = 0;

SxValue* __native_http_request(SxValue *method, SxValue *url,
                                SxValue *headers, SxValue *body) {
    expect_str(method, "http_request");
    expect_str(url, "http_request");

    if (!_curl_initialized) {
        curl_global_init(CURL_GLOBAL_DEFAULT);
        _curl_initialized = 1;
    }

    CURL *curl = curl_easy_init();
    if (!curl) return sx_error_new(sx_string("Failed to init HTTP client"));

    CurlBuffer buf = {NULL, 0, 0};

    curl_easy_setopt(curl, CURLOPT_URL, url->string);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curl_write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &buf);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);

    if (strcmp(method->string, "POST") == 0) {
        curl_easy_setopt(curl, CURLOPT_POST, 1L);
        if (body && body->type == SX_STRING) {
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body->string);
            curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, (long)strlen(body->string));
        }
    } else if (strcmp(method->string, "PUT") == 0) {
        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");
        if (body && body->type == SX_STRING)
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body->string);
    } else if (strcmp(method->string, "DELETE") == 0) {
        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "DELETE");
    }

    struct curl_slist *hlist = NULL;
    if (headers && headers->type == SX_DICT) {
        for (int i = 0; i < headers->dict.len; i++) {
            SxDictBucket *b = sx_dict_find(&headers->dict, headers->dict.keys[i]);
            if (b && b->value && b->value->type == SX_STRING) {
                char hdr[512];
                snprintf(hdr, sizeof(hdr), "%s: %s", headers->dict.keys[i], b->value->string);
                hlist = curl_slist_append(hlist, hdr);
            }
        }
        if (hlist) curl_easy_setopt(curl, CURLOPT_HTTPHEADER, hlist);
    }

    CURLcode res = curl_easy_perform(curl);
    if (hlist) curl_slist_free_all(hlist);

    if (res != CURLE_OK) {
        SxValue *err = sx_error_new(sx_string(curl_easy_strerror(res)));
        curl_easy_cleanup(curl);
        if (buf.data) SX_FREE(buf.data);
        return err;
    }

    curl_easy_cleanup(curl);

    if (!buf.data) return sx_string("");
    buf.data = (char *)SX_REALLOC(buf.data, buf.len + 1);
    buf.data[buf.len] = '\0';

    SxValue *r = sx_alloc(SX_STRING);
    r->string = buf.data;
    return r;
}

#else

/* Stub when libcurl is not available */
SxValue* __native_http_request(SxValue *method, SxValue *url,
                                SxValue *headers, SxValue *body) {
    (void)method; (void)url; (void)headers; (void)body;
    return sx_error_new(sx_string("HTTP not available (compile with -DSX_USE_CURL -lcurl)"));
}

#endif
