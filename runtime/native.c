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

/* --- Math natives --- */

SxValue* __native_sqrt(SxValue *v)  { return sx_number(sqrt(v->number)); }
SxValue* __native_sin(SxValue *v)   { return sx_number(sin(v->number)); }
SxValue* __native_cos(SxValue *v)   { return sx_number(cos(v->number)); }
SxValue* __native_tan(SxValue *v)   { return sx_number(tan(v->number)); }
SxValue* __native_asin(SxValue *v)  { return sx_number(asin(v->number)); }
SxValue* __native_acos(SxValue *v)  { return sx_number(acos(v->number)); }
SxValue* __native_atan(SxValue *v)  { return sx_number(atan(v->number)); }
SxValue* __native_log(SxValue *v)   { return sx_number(log(v->number)); }
SxValue* __native_log2(SxValue *v)  { return sx_number(log2(v->number)); }
SxValue* __native_log10(SxValue *v) { return sx_number(log10(v->number)); }
SxValue* __native_exp(SxValue *v)   { return sx_number(exp(v->number)); }
SxValue* __native_floor(SxValue *v) { return sx_number(floor(v->number)); }
SxValue* __native_ceil(SxValue *v)  { return sx_number(ceil(v->number)); }
SxValue* __native_round(SxValue *v) { return sx_number(round(v->number)); }
SxValue* __native_cbrt(SxValue *v)  { return sx_number(cbrt(v->number)); }
SxValue* __native_pow(SxValue *a, SxValue *b) { return sx_number(pow(a->number, b->number)); }

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
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = toupper((unsigned char)s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* __native_lower(SxValue *v) {
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = tolower((unsigned char)s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* __native_split(SxValue *v, SxValue *sep) {
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
    FILE *f = fopen(path->string, "r");
    if (!f) return sx_error_new("Cannot read file");

    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    if (len < 0) {
        // Non-seekable stream or error — read in chunks
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
    FILE *f = fopen(path->string, "w");
    if (!f) return sx_error_new("Cannot write file");
    SxValue *s = sx_to_string(data);
    fputs(s->string, f);
    fclose(f);
    return sx_null();
}

SxValue* __native_file_exists(SxValue *path) {
    struct stat st;
    return sx_bool(stat(path->string, &st) == 0);
}

SxValue* __native_delete_file(SxValue *path) {
    if (remove(path->string) != 0) return sx_error_new("Cannot delete file");
    return sx_null();
}

SxValue* __native_cwd(void) {
    char buf[4096];
    if (getcwd(buf, sizeof(buf))) return sx_string(buf);
    return sx_string(".");
}

SxValue* __native_getenv(SxValue *name) {
    char *val = getenv(name->string);
    if (!val) return sx_null();
    return sx_string(val);
}

SxValue* __native_exec(SxValue *cmd) {
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
