#include "runtime.h"
#include <string.h>
#include <ctype.h>

SxValue* sx_string_upper(SxValue *v) {
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = toupper(s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* sx_string_lower(SxValue *v) {
    char *s = sx_strdup(v->string);
    for (int i = 0; s[i]; i++) s[i] = tolower(s[i]);
    SxValue *r = sx_alloc(SX_STRING); r->string = s; return r;
}

SxValue* sx_string_trim(SxValue *v) {
    char *s = v->string;
    while (*s && (*s == ' ' || *s == '\t' || *s == '\n' || *s == '\r')) s++;
    char *end = s + strlen(s) - 1;
    while (end > s && (*end == ' ' || *end == '\t' || *end == '\n' || *end == '\r')) end--;
    int len = end - s + 1;
    char *result = (char*)SX_MALLOC(len + 1);
    memcpy(result, s, len);
    result[len] = '\0';
    SxValue *r = sx_alloc(SX_STRING); r->string = result; return r;
}

SxValue* sx_string_split(SxValue *v, SxValue *sep) {
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
    return list;
}

SxValue* sx_string_replace(SxValue *v, SxValue *old, SxValue *new_) {
    char *s = v->string;
    char *o = old->string;
    char *n = new_->string;
    int olen = strlen(o), nlen = strlen(n), slen = strlen(s);

    // Count occurrences
    int count = 0;
    char *p = s;
    while ((p = strstr(p, o)) != NULL) { count++; p += olen; }

    char *result = (char*)SX_MALLOC(slen + count * (nlen - olen) + 1);
    char *dst = result;
    p = s;
    char *found;
    while ((found = strstr(p, o)) != NULL) {
        int chunk = found - p;
        memcpy(dst, p, chunk);
        dst += chunk;
        memcpy(dst, n, nlen);
        dst += nlen;
        p = found + olen;
    }
    strcpy(dst, p);
    SxValue *r = sx_alloc(SX_STRING); r->string = result; return r;
}

SxValue* sx_string_contains(SxValue *v, SxValue *sub) {
    return sx_bool(strstr(v->string, sub->string) != NULL);
}

SxValue* sx_string_starts_with(SxValue *v, SxValue *prefix) {
    return sx_bool(strncmp(v->string, prefix->string, strlen(prefix->string)) == 0);
}

SxValue* sx_string_ends_with(SxValue *v, SxValue *suffix) {
    int slen = strlen(v->string), plen = strlen(suffix->string);
    if (plen > slen) return sx_bool(0);
    return sx_bool(strcmp(v->string + slen - plen, suffix->string) == 0);
}

SxValue* sx_string_join(SxValue *list, SxValue *sep) {
    int total = 0;
    for (int i = 0; i < list->list.len; i++) {
        SxValue *s = sx_to_string(list->list.items[i]);
        total += strlen(s->string);
        if (i > 0) total += strlen(sep->string);
    }
    char *result = (char*)SX_MALLOC(total + 1);
    int offset = 0;
    for (int i = 0; i < list->list.len; i++) {
        if (i > 0) {
            int slen = strlen(sep->string);
            memcpy(result + offset, sep->string, slen);
            offset += slen;
        }
        SxValue *s = sx_to_string(list->list.items[i]);
        int len = strlen(s->string);
        memcpy(result + offset, s->string, len);
        offset += len;
    }
    result[offset] = '\0';
    SxValue *r = sx_alloc(SX_STRING); r->string = result; return r;
}
