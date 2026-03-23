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
    int olen = strlen(o), nlen = strlen(n), slen = strlen(s);
    if (olen == 0) return sx_string(s);

    int count = 0;
    char *p = s;
    while ((p = strstr(p, o)) != NULL) { count++; p += olen; }

    char *result = (char*)SX_MALLOC(slen + count * (nlen - olen) + 1);
    char *dst = result;
    p = s;
    char *found;
    while ((found = strstr(p, o)) != NULL) {
        int chunk = found - p;
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
    if (!f) return sx_error_new("Cannot read file");

    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    if (len < 0) {
        fclose(f);
        f = fopen(path->string, "r");
        if (!f) return sx_error_new("Cannot read file");
        size_t cap = 4096, total = 0;
        char *buf = (char*)SX_MALLOC(cap);
        size_t n;
        while ((n = fread(buf + total, 1, cap - total, f)) > 0) {
            total += n;
            if (total >= cap) {
                cap *= 2;
                buf = (char*)SX_REALLOC(buf, cap);
            }
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
    if (!f) return sx_error_new("Cannot write file");
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
    if (remove(path->string) != 0) return sx_error_new("Cannot delete file");
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
