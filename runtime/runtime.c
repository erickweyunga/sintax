#include "runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <math.h>
#include <ctype.h>

// --- Memory helpers ---

SxValue* sx_alloc(SxType type) {
    SxValue *v = (SxValue*)SX_MALLOC(sizeof(SxValue));
    if (!v) { fprintf(stderr, "Error: out of memory\n"); exit(1); }
    v->type = type;
    return v;
}

char* sx_strdup(const char *s) {
    size_t len = strlen(s);
    char *dup = (char*)SX_MALLOC(len + 1);
    if (!dup) { fprintf(stderr, "Error: out of memory\n"); exit(1); }
    memcpy(dup, s, len + 1);
    return dup;
}

// --- Singletons (avoid allocation for common values) ---

static SxValue _sx_null_singleton = {.type = SX_NULL};
static SxValue _sx_true_singleton = {.type = SX_BOOL, .boolean = 1};
static SxValue _sx_false_singleton = {.type = SX_BOOL, .boolean = 0};

// Small integer cache (0-255)
#define SX_INT_CACHE_SIZE 256
static SxValue _sx_int_cache[SX_INT_CACHE_SIZE];
static int _sx_int_cache_init = 0;

static void sx_init_int_cache(void) {
    if (_sx_int_cache_init) return;
    for (int i = 0; i < SX_INT_CACHE_SIZE; i++) {
        _sx_int_cache[i].type = SX_NUMBER;
        _sx_int_cache[i].number = (double)i;
    }
    _sx_int_cache_init = 1;
}

// --- Constructors ---

SxValue* sx_number(double n) {
    if (n >= 0 && n < SX_INT_CACHE_SIZE && n == (int)n) {
        sx_init_int_cache();
        return &_sx_int_cache[(int)n];
    }
    SxValue *v = sx_alloc(SX_NUMBER);
    v->number = n;
    return v;
}

SxValue* sx_string(const char *s) {
    SxValue *v = sx_alloc(SX_STRING);
    v->string = sx_strdup(s);
    return v;
}

SxValue* sx_bool(int b) {
    return b ? &_sx_true_singleton : &_sx_false_singleton;
}

SxValue* sx_null(void) {
    return &_sx_null_singleton;
}

SxValue* sx_list_new(void) {
    SxValue *v = sx_alloc(SX_LIST);
    v->list.items = NULL;
    v->list.len = 0;
    v->list.cap = 0;
    return v;
}

#define SX_DICT_INIT_CAP 8
#define SX_DICT_LOAD_FACTOR 0.75

static unsigned long sx_hash(const char *s) {
    unsigned long hash = 5381;
    int c;
    while ((c = *s++))
        hash = ((hash << 5) + hash) + c; // djb2
    return hash;
}

SxValue* sx_dict_new(void) {
    SxValue *v = sx_alloc(SX_DICT);
    v->dict.capacity = SX_DICT_INIT_CAP;
    v->dict.buckets = (SxDictBucket*)SX_MALLOC(sizeof(SxDictBucket) * SX_DICT_INIT_CAP);
    memset(v->dict.buckets, 0, sizeof(SxDictBucket) * SX_DICT_INIT_CAP);
    v->dict.len = 0;
    v->dict.keys = NULL;
    v->dict.keys_cap = 0;
    return v;
}

static void sx_dict_resize(SxDict *d) {
    int new_cap = d->capacity * 2;
    SxDictBucket *new_buckets = (SxDictBucket*)SX_MALLOC(sizeof(SxDictBucket) * new_cap);
    memset(new_buckets, 0, sizeof(SxDictBucket) * new_cap);

    for (int i = 0; i < d->capacity; i++) {
        if (d->buckets[i].used) {
            unsigned long h = sx_hash(d->buckets[i].key) & (new_cap - 1);
            while (new_buckets[h].used)
                h = (h + 1) & (new_cap - 1);
            new_buckets[h] = d->buckets[i];
        }
    }

    SX_FREE(d->buckets);
    d->buckets = new_buckets;
    d->capacity = new_cap;
}

SxDictBucket* sx_dict_find(SxDict *d, const char *key) {
    unsigned long h = sx_hash(key) & (d->capacity - 1);
    int attempts = 0;
    while (d->buckets[h].used) {
        if (strcmp(d->buckets[h].key, key) == 0)
            return &d->buckets[h];
        h = (h + 1) & (d->capacity - 1);
        if (++attempts >= d->capacity) break; // prevent infinite loop
    }
    return NULL;
}

static void sx_dict_add_key(SxDict *d, const char *key) {
    if (d->len >= d->keys_cap) {
        d->keys_cap = d->keys_cap == 0 ? 4 : d->keys_cap * 2;
        d->keys = (char**)SX_REALLOC(d->keys, sizeof(char*) * d->keys_cap);
    }
    d->keys[d->len] = sx_strdup(key);
}

// --- Function / Error constructors ---

SxValue* sx_function(SxFnPtr fn) {
    SxValue *v = sx_alloc(SX_FUNCTION);
    v->function = fn;
    return v;
}

SxValue* sx_error_new(const char *msg) {
    SxValue *v = sx_alloc(SX_ERROR);
    v->string = sx_strdup(msg);
    return v;
}

SxValue* sx_is_error(SxValue *v) {
    return sx_bool(v && v->type == SX_ERROR);
}

SxValue* sx_call(SxValue *fn, SxValue **args, int argc) {
    if (!fn || fn->type != SX_FUNCTION) {
        sx_error("Not a function");
    }
    return fn->function(args, argc);
}

void sx_error(const char *msg) {
    fprintf(stderr, "Error: %s\n", msg);
    exit(1);
}

// --- String replace helper (used by sx_method and __native_replace) ---

static SxValue* sx_string_replace_impl(const char *s, const char *old, const char *new_) {
    int olen = strlen(old), nlen = strlen(new_), slen = strlen(s);
    if (olen == 0) return sx_string(s);

    // Count occurrences
    int count = 0;
    const char *p = s;
    while ((p = strstr(p, old)) != NULL) { count++; p += olen; }

    // Build result
    char *result = (char*)SX_MALLOC(slen + count * (nlen - olen) + 1);
    char *dst = result;
    p = s;
    const char *found;
    while ((found = strstr(p, old)) != NULL) {
        int chunk = found - p;
        memcpy(dst, p, chunk); dst += chunk;
        memcpy(dst, new_, nlen); dst += nlen;
        p = found + olen;
    }
    strcpy(dst, p);

    SxValue *v = sx_alloc(SX_STRING);
    v->string = result;
    return v;
}

// --- Method dispatch ---

SxValue* sx_method(SxValue *obj, const char *name, SxValue **args, int argc) {
    // String methods
    if (obj->type == SX_STRING) {
        if (strcmp(name, "len") == 0) return sx_number(strlen(obj->string));
        if (strcmp(name, "upper") == 0) {
            char *s = sx_strdup(obj->string);
            for (int i = 0; s[i]; i++) s[i] = toupper(s[i]);
            SxValue *v = sx_alloc(SX_STRING); v->string = s; return v;
        }
        if (strcmp(name, "lower") == 0) {
            char *s = sx_strdup(obj->string);
            for (int i = 0; s[i]; i++) s[i] = tolower(s[i]);
            SxValue *v = sx_alloc(SX_STRING); v->string = s; return v;
        }
        if (strcmp(name, "trim") == 0) {
            const char *s = obj->string;
            while (*s && isspace((unsigned char)*s)) s++;
            const char *end = s + strlen(s);
            while (end > s && isspace((unsigned char)*(end - 1))) end--;
            int len = end - s;
            char *result = (char*)SX_MALLOC(len + 1);
            memcpy(result, s, len);
            result[len] = '\0';
            SxValue *v = sx_alloc(SX_STRING); v->string = result; return v;
        }
        if (strcmp(name, "contains") == 0 && argc == 1) {
            if (args[0]->type != SX_STRING) sx_error("str.contains() requires a str argument");
            return sx_bool(strstr(obj->string, args[0]->string) != NULL);
        }
        if (strcmp(name, "starts_with") == 0 && argc == 1) {
            if (args[0]->type != SX_STRING) sx_error("str.starts_with() requires a str argument");
            size_t plen = strlen(args[0]->string);
            return sx_bool(strncmp(obj->string, args[0]->string, plen) == 0);
        }
        if (strcmp(name, "ends_with") == 0 && argc == 1) {
            if (args[0]->type != SX_STRING) sx_error("str.ends_with() requires a str argument");
            size_t slen = strlen(obj->string);
            size_t sufflen = strlen(args[0]->string);
            if (sufflen > slen) return sx_bool(0);
            return sx_bool(strcmp(obj->string + slen - sufflen, args[0]->string) == 0);
        }
        if (strcmp(name, "split") == 0 && argc == 1) {
            if (args[0]->type != SX_STRING) sx_error("str.split() requires a str argument");
            SxValue *list = sx_list_new();
            char *s = sx_strdup(obj->string);
            char *sep = args[0]->string;
            int sep_len = strlen(sep);
            char *start = s;
            char *found;
            while ((found = strstr(start, sep)) != NULL) {
                *found = '\0';
                sx_list_append(list, sx_string(start));
                start = found + sep_len;
            }
            sx_list_append(list, sx_string(start));
            SX_FREE(s);
            return list;
        }
        if (strcmp(name, "replace") == 0 && argc == 2) {
            if (args[0]->type != SX_STRING || args[1]->type != SX_STRING)
                sx_error("str.replace() requires str arguments");
            return sx_string_replace_impl(obj->string, args[0]->string, args[1]->string);
        }
        if (strcmp(name, "type") == 0) return sx_string("str");
    }

    // List methods
    if (obj->type == SX_LIST) {
        if (strcmp(name, "len") == 0) return sx_number(obj->list.len);
        if (strcmp(name, "push") == 0 && argc == 1) { sx_list_append(obj, args[0]); return obj; }
        if (strcmp(name, "pop") == 0 && argc == 1) { return sx_list_remove(obj, args[0]); }
        if (strcmp(name, "contains") == 0 && argc == 1) {
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *eq = sx_eq(args[0], obj->list.items[i]);
                if (eq->boolean) return sx_bool(1);
            }
            return sx_bool(0);
        }
        if (strcmp(name, "reverse") == 0) {
            SxValue *list = sx_list_new();
            for (int i = obj->list.len - 1; i >= 0; i--)
                sx_list_append(list, obj->list.items[i]);
            return list;
        }
        if (strcmp(name, "join") == 0 && argc == 1) {
            if (args[0]->type != SX_STRING) sx_error("list.join() requires a str argument");
            char *sep = args[0]->string;
            int sep_len = strlen(sep);
            // Calculate total length
            int total = 0;
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *s = sx_to_string(obj->list.items[i]);
                total += strlen(s->string);
                if (i > 0) total += sep_len;
            }
            char *result = (char*)SX_MALLOC(total + 1);
            int offset = 0;
            for (int i = 0; i < obj->list.len; i++) {
                if (i > 0) { memcpy(result + offset, sep, sep_len); offset += sep_len; }
                SxValue *s = sx_to_string(obj->list.items[i]);
                int len = strlen(s->string);
                memcpy(result + offset, s->string, len);
                offset += len;
            }
            result[offset] = '\0';
            SxValue *v = sx_alloc(SX_STRING); v->string = result; return v;
        }
        if (strcmp(name, "map") == 0 && argc == 1) {
            if (args[0]->type != SX_FUNCTION) sx_error("list.map() argument must be a function");
            SxValue *list = sx_list_new();
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *item = obj->list.items[i];
                sx_list_append(list, args[0]->function(&item, 1));
            }
            return list;
        }
        if (strcmp(name, "filter") == 0 && argc == 1) {
            if (args[0]->type != SX_FUNCTION) sx_error("list.filter() argument must be a function");
            SxValue *list = sx_list_new();
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *item = obj->list.items[i];
                if (sx_truthy(args[0]->function(&item, 1)))
                    sx_list_append(list, item);
            }
            return list;
        }
        if (strcmp(name, "reduce") == 0 && argc == 2) {
            if (args[0]->type != SX_FUNCTION) sx_error("list.reduce() first argument must be a function");
            SxValue *acc = args[1];
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *pair[2] = {acc, obj->list.items[i]};
                acc = args[0]->function(pair, 2);
            }
            return acc;
        }
        if (strcmp(name, "each") == 0 && argc == 1) {
            if (args[0]->type != SX_FUNCTION) sx_error("list.each() argument must be a function");
            for (int i = 0; i < obj->list.len; i++) {
                SxValue *item = obj->list.items[i];
                args[0]->function(&item, 1);
            }
            return sx_null();
        }
        if (strcmp(name, "type") == 0) return sx_string("list");
    }

    // Dict methods
    if (obj->type == SX_DICT) {
        if (strcmp(name, "len") == 0) return sx_number(obj->dict.len);
        if (strcmp(name, "keys") == 0) return sx_dict_keys(obj);
        if (strcmp(name, "values") == 0) return sx_dict_values(obj);
        if (strcmp(name, "has") == 0 && argc == 1) return sx_dict_has(obj, args[0]);
        if (strcmp(name, "type") == 0) return sx_string("dict");
    }

    // Num/Bool methods
    if (obj->type == SX_NUMBER) {
        if (strcmp(name, "type") == 0) return sx_string("num");
    }
    if (obj->type == SX_BOOL) {
        if (strcmp(name, "type") == 0) return sx_string("bool");
    }

    char msg[256];
    snprintf(msg, sizeof(msg), "'%s' has no method '%s'", sx_type(obj)->string, name);
    sx_error(msg);
    return sx_null();
}

// --- Truthy ---

int sx_truthy(SxValue *a) {
    if (!a) return 0;
    switch (a->type) {
        case SX_NULL: return 0;
        case SX_ERROR: return 0;
        case SX_BOOL: return a->boolean;
        case SX_NUMBER: return a->number != 0;
        case SX_STRING: return a->string[0] != '\0';
        case SX_LIST: return a->list.len > 0;
        case SX_DICT: return a->dict.len > 0;
        default: return 1;
    }
}

// --- Arithmetic ---

SxValue* sx_add(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(a->number + b->number);
    if (a->type == SX_STRING && b->type == SX_STRING) {
        size_t la = strlen(a->string), lb = strlen(b->string);
        char *s = (char*)SX_MALLOC(la + lb + 1);
        memcpy(s, a->string, la);
        memcpy(s + la, b->string, lb + 1);
        SxValue *v = sx_alloc(SX_STRING);
        v->string = s;
        return v;
    }
    sx_error("Operation '+' not supported for these types");
    return sx_null();
}

SxValue* sx_sub(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(a->number - b->number);
    sx_error("Operation '-' requires num values");
    return sx_null();
}

SxValue* sx_mul(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(a->number * b->number);
    sx_error("Operation '*' requires num values");
    return sx_null();
}

SxValue* sx_div(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER) {
        if (b->number == 0) sx_error("Division by zero");
        return sx_number(a->number / b->number);
    }
    sx_error("Operation '/' requires num values");
    return sx_null();
}

SxValue* sx_mod(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER) {
        if (b->number == 0) sx_error("Division by zero");
        return sx_number((double)((long long)a->number % (long long)b->number));
    }
    sx_error("Operation '%%' requires num values");
    return sx_null();
}

SxValue* sx_pow(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(pow(a->number, b->number));
    sx_error("Operation '**' requires num values");
    return sx_null();
}

// --- Comparison ---

SxValue* sx_eq(SxValue *a, SxValue *b) {
    if (a->type != b->type) return sx_bool(0);
    switch (a->type) {
        case SX_NUMBER: return sx_bool(a->number == b->number);
        case SX_STRING: return sx_bool(strcmp(a->string, b->string) == 0);
        case SX_BOOL: return sx_bool(a->boolean == b->boolean);
        case SX_NULL: return sx_bool(1);
        default: return sx_bool(0);
    }
}

SxValue* sx_neq(SxValue *a, SxValue *b) {
    SxValue *eq = sx_eq(a, b);
    return sx_bool(!eq->boolean);
}

SxValue* sx_gt(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number > b->number);
    sx_error("Comparison '>' requires num values");
    return sx_null();
}

SxValue* sx_lt(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number < b->number);
    sx_error("Comparison '<' requires num values");
    return sx_null();
}

SxValue* sx_gte(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number >= b->number);
    sx_error("Comparison '>=' requires num values");
    return sx_null();
}

SxValue* sx_lte(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number <= b->number);
    sx_error("Comparison '<=' requires num values");
    return sx_null();
}

// --- Logical ---

SxValue* sx_not(SxValue *a) {
    return sx_bool(!sx_truthy(a));
}

// --- Print ---

static void sx_print_value(SxValue *v, int quote_strings) {
    if (!v) { printf("null"); return; }
    switch (v->type) {
        case SX_NULL: printf("null"); break;
        case SX_NUMBER:
            if (v->number == (long long)v->number)
                printf("%lld", (long long)v->number);
            else
                printf("%g", v->number);
            break;
        case SX_STRING:
            if (quote_strings) printf("\"%s\"", v->string);
            else printf("%s", v->string);
            break;
        case SX_BOOL: printf("%s", v->boolean ? "true" : "false"); break;
        case SX_LIST:
            printf("[");
            for (int i = 0; i < v->list.len; i++) {
                if (i > 0) printf(", ");
                sx_print_value(v->list.items[i], 1);
            }
            printf("]");
            break;
        case SX_DICT:
            printf("{");
            for (int i = 0; i < v->dict.len; i++) {
                if (i > 0) printf(", ");
                printf("\"%s\": ", v->dict.keys[i]);
                SxDictBucket *b = sx_dict_find(&v->dict, v->dict.keys[i]);
                if (b) sx_print_value(b->value, 1);
            }
            printf("}");
            break;
        case SX_FUNCTION: printf("<fn>"); break;
        case SX_ERROR: printf("error: %s", v->string); break;
    }
}

void sx_print(SxValue *v) {
    sx_print_value(v, 0);
    printf("\n");
}

void sx_print_multi(int count, ...) {
    va_list args;
    va_start(args, count);
    for (int i = 0; i < count; i++) {
        if (i > 0) printf(" ");
        SxValue *v = va_arg(args, SxValue*);
        sx_print_value(v, 0);
    }
    va_end(args);
    printf("\n");
}

// --- List operations ---

void sx_list_append(SxValue *list, SxValue *item) {
    if (list->type != SX_LIST) sx_error("push() first argument must be a list");
    if (list->list.len >= list->list.cap) {
        list->list.cap = list->list.cap == 0 ? 4 : list->list.cap * 2;
        list->list.items = (SxValue**)SX_REALLOC(list->list.items, sizeof(SxValue*) * list->list.cap);
    }
    list->list.items[list->list.len++] = item;
}

SxValue* sx_list_get(SxValue *list, SxValue *index) {
    if (list->type != SX_LIST) sx_error("not a list");
    if (index->type != SX_NUMBER) sx_error("Index must be a num");
    int i = (int)index->number;
    if (i < 0) i += list->list.len;
    if (i < 0 || i >= list->list.len) sx_error("Index out of range");
    return list->list.items[i];
}

void sx_list_set(SxValue *list, SxValue *index, SxValue *val) {
    if (list->type != SX_LIST) sx_error("not a list");
    if (index->type != SX_NUMBER) sx_error("Index must be a num");
    int i = (int)index->number;
    if (i < 0) i += list->list.len;
    if (i < 0 || i >= list->list.len) sx_error("Index out of range");
    list->list.items[i] = val;
}

SxValue* sx_list_remove(SxValue *list, SxValue *index) {
    if (list->type != SX_LIST) sx_error("pop() first argument must be a list");
    if (index->type != SX_NUMBER) sx_error("pop() second argument must be a num");
    int i = (int)index->number;
    if (i < 0 || i >= list->list.len) sx_error("Index out of range");
    SxValue *removed = list->list.items[i];
    for (int j = i; j < list->list.len - 1; j++)
        list->list.items[j] = list->list.items[j + 1];
    list->list.len--;
    return removed;
}

// --- Dict operations (hash table) ---

SxValue* sx_dict_get(SxValue *dict, SxValue *key) {
    if (dict->type != SX_DICT) sx_error("not a dict");
    if (key->type != SX_STRING) sx_error("Dict key must be a str");
    SxDictBucket *b = sx_dict_find(&dict->dict, key->string);
    return b ? b->value : sx_null();
}

void sx_dict_set(SxValue *dict, SxValue *key, SxValue *val) {
    if (dict->type != SX_DICT) sx_error("not a dict");
    if (key->type != SX_STRING) sx_error("Dict key must be a str");

    SxDictBucket *existing = sx_dict_find(&dict->dict, key->string);
    if (existing) {
        existing->value = val;
        return;
    }

    if ((double)dict->dict.len / dict->dict.capacity >= SX_DICT_LOAD_FACTOR) {
        sx_dict_resize(&dict->dict);
    }

    unsigned long h = sx_hash(key->string) & (dict->dict.capacity - 1);
    while (dict->dict.buckets[h].used)
        h = (h + 1) & (dict->dict.capacity - 1);

    dict->dict.buckets[h].key = sx_strdup(key->string);
    dict->dict.buckets[h].value = val;
    dict->dict.buckets[h].used = 1;

    sx_dict_add_key(&dict->dict, key->string);
    dict->dict.len++;
}

SxValue* sx_dict_keys(SxValue *dict) {
    if (dict->type != SX_DICT) sx_error("keys() requires a dict");
    SxValue *list = sx_list_new();
    for (int i = 0; i < dict->dict.len; i++)
        sx_list_append(list, sx_string(dict->dict.keys[i]));
    return list;
}

SxValue* sx_dict_values(SxValue *dict) {
    if (dict->type != SX_DICT) sx_error("values() requires a dict");
    SxValue *list = sx_list_new();
    for (int i = 0; i < dict->dict.len; i++) {
        SxDictBucket *b = sx_dict_find(&dict->dict, dict->dict.keys[i]);
        if (b) sx_list_append(list, b->value);
    }
    return list;
}

SxValue* sx_dict_has(SxValue *dict, SxValue *key) {
    if (dict->type != SX_DICT) sx_error("has() first argument must be a dict");
    if (key->type != SX_STRING) sx_error("has() second argument must be a str");
    return sx_bool(sx_dict_find(&dict->dict, key->string) != NULL);
}

// --- Generic index ---

SxValue* sx_index(SxValue *collection, SxValue *idx) {
    switch (collection->type) {
        case SX_LIST: return sx_list_get(collection, idx);
        case SX_DICT: return sx_dict_get(collection, idx);
        case SX_STRING: {
            if (idx->type != SX_NUMBER) sx_error("Index must be a num");
            int slen = (int)strlen(collection->string);
            int i = (int)idx->number;
            if (i < 0) i += slen;
            if (i < 0 || i >= slen) sx_error("Index out of range");
            char buf[2] = {collection->string[i], '\0'};
            return sx_string(buf);
        }
        default: sx_error("Not indexable");
    }
    return sx_null();
}

void sx_index_set(SxValue *collection, SxValue *idx, SxValue *val) {
    switch (collection->type) {
        case SX_LIST: sx_list_set(collection, idx, val); break;
        case SX_DICT: sx_dict_set(collection, idx, val); break;
        default: sx_error("Cannot assign by index");
    }
}

// --- Membership ---

SxValue* sx_in(SxValue *needle, SxValue *haystack) {
    switch (haystack->type) {
        case SX_LIST:
            for (int i = 0; i < haystack->list.len; i++) {
                SxValue *eq = sx_eq(needle, haystack->list.items[i]);
                if (eq->boolean) return sx_bool(1);
            }
            return sx_bool(0);
        case SX_DICT:
            return sx_dict_has(haystack, needle);
        case SX_STRING:
            if (needle->type != SX_STRING) sx_error("'in' for str requires a str");
            return sx_bool(strstr(haystack->string, needle->string) != NULL);
        default:
            sx_error("'in' not supported for this type");
    }
    return sx_bool(0);
}

// --- Utilities ---

SxValue* sx_len(SxValue *v) {
    switch (v->type) {
        case SX_STRING: return sx_number(strlen(v->string));
        case SX_LIST: return sx_number(v->list.len);
        case SX_DICT: return sx_number(v->dict.len);
        default: sx_error("len() requires a list, str, or dict");
    }
    return sx_null();
}

SxValue* sx_type(SxValue *v) {
    switch (v->type) {
        case SX_NUMBER: return sx_string("num");
        case SX_STRING: return sx_string("str");
        case SX_BOOL: return sx_string("bool");
        case SX_LIST: return sx_string("list");
        case SX_DICT: return sx_string("dict");
        case SX_NULL: return sx_string("null");
        case SX_FUNCTION: return sx_string("fn");
        case SX_ERROR: return sx_string("error");
        default: return sx_string("unknown");
    }
}

SxValue* sx_range(SxValue *start, SxValue *end) {
    if (start->type != SX_NUMBER || end->type != SX_NUMBER)
        sx_error("range() requires num arguments");
    SxValue *list = sx_list_new();
    for (double i = start->number; i < end->number; i++)
        sx_list_append(list, sx_number(i));
    return list;
}

SxValue* sx_to_number(SxValue *v) {
    switch (v->type) {
        case SX_NUMBER: return v;
        case SX_STRING: {
            char *end;
            double n = strtod(v->string, &end);
            if (end == v->string) sx_error("Cannot convert str to num");
            return sx_number(n);
        }
        case SX_BOOL: return sx_number(v->boolean ? 1 : 0);
        default: sx_error("Cannot convert to num");
    }
    return sx_null();
}

SxValue* sx_to_string(SxValue *v) {
    char buf[256];
    switch (v->type) {
        case SX_NUMBER:
            if (v->number == (long long)v->number)
                snprintf(buf, sizeof(buf), "%lld", (long long)v->number);
            else
                snprintf(buf, sizeof(buf), "%g", v->number);
            return sx_string(buf);
        case SX_STRING: return v;
        case SX_BOOL: return sx_string(v->boolean ? "true" : "false");
        case SX_NULL: return sx_string("null");
        case SX_ERROR: return sx_string(v->string);
        default: return sx_string("<object>");
    }
}

SxValue* sx_to_bool(SxValue *v) {
    return sx_bool(sx_truthy(v));
}

SxValue* sx_concat(int count, ...) {
    va_list args;

    // Convert all values to strings once
    SxValue **strs = (SxValue**)SX_MALLOC(sizeof(SxValue*) * count);
    size_t total = 0;
    va_start(args, count);
    for (int i = 0; i < count; i++) {
        SxValue *v = va_arg(args, SxValue*);
        strs[i] = sx_to_string(v);
        total += strlen(strs[i]->string);
    }
    va_end(args);

    // Build result in one pass
    char *result = (char*)SX_MALLOC(total + 1);
    size_t offset = 0;
    for (int i = 0; i < count; i++) {
        size_t len = strlen(strs[i]->string);
        memcpy(result + offset, strs[i]->string, len);
        offset += len;
    }
    result[offset] = '\0';
    SX_FREE(strs);

    SxValue *sv = sx_alloc(SX_STRING);
    sv->string = result;
    return sv;
}

void sx_check_type(SxValue *v, SxType expected, const char *name) {
    if (v->type != expected) {
        char msg[256];
        SxValue *got = sx_type(v);
        const char *exp_name;
        switch (expected) {
            case SX_NUMBER: exp_name = "num"; break;
            case SX_STRING: exp_name = "str"; break;
            case SX_BOOL: exp_name = "bool"; break;
            case SX_LIST: exp_name = "list"; break;
            case SX_DICT: exp_name = "dict"; break;
            default: exp_name = "unknown"; break;
        }
        snprintf(msg, sizeof(msg), "Type mismatch: '%s' is %s, expected %s", name, got->string, exp_name);
        sx_error(msg);
    }
}

// --- Input ---
SxValue* sx_input(SxValue *prompt) {
    if (prompt && prompt->type == SX_STRING)
        printf("%s", prompt->string);
    char buf[4096];
    if (!fgets(buf, sizeof(buf), stdin)) return sx_string("");
    buf[strcspn(buf, "\r\n")] = '\0';
    // Try to parse as number
    char *end;
    double n = strtod(buf, &end);
    if (*end == '\0' && end != buf) return sx_number(n);
    return sx_string(buf);
}
