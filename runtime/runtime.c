#include "runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <math.h>

#ifdef SX_USE_GC
#include <gc.h>
#define SX_MALLOC(size) GC_MALLOC(size)
#define SX_REALLOC(ptr, size) GC_REALLOC(ptr, size)
#define SX_FREE(ptr) /* GC handles it */
#else
#define SX_MALLOC(size) malloc(size)
#define SX_REALLOC(ptr, size) realloc(ptr, size)
#define SX_FREE(ptr) free(ptr)
#endif

// --- Memory helpers ---

static SxValue* sx_alloc(SxType type) {
    SxValue *v = (SxValue*)SX_MALLOC(sizeof(SxValue));
    if (!v) { fprintf(stderr, "Kosa: kumbukumbu imekwisha\n"); exit(1); }
    v->type = type;
    return v;
}

static char* sx_strdup(const char *s) {
    size_t len = strlen(s);
    char *dup = (char*)SX_MALLOC(len + 1);
    if (!dup) { fprintf(stderr, "Kosa: kumbukumbu imekwisha\n"); exit(1); }
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
    // Use cache for small integers
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

    // Rehash all entries
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

static SxDictBucket* sx_dict_find(SxDict *d, const char *key) {
    unsigned long h = sx_hash(key) & (d->capacity - 1);
    while (d->buckets[h].used) {
        if (strcmp(d->buckets[h].key, key) == 0)
            return &d->buckets[h];
        h = (h + 1) & (d->capacity - 1);
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

// --- Error ---

void sx_error(const char *msg) {
    fprintf(stderr, "Kosa: %s\n", msg);
    exit(1);
}

// --- Truthy ---

int sx_truthy(SxValue *a) {
    if (!a) return 0;
    switch (a->type) {
        case SX_NULL: return 0;
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
    sx_error("Operesheni '+' haiwezekani kwa aina hizi");
    return sx_null();
}

SxValue* sx_sub(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(a->number - b->number);
    sx_error("Operesheni '-' inahitaji nambari");
    return sx_null();
}

SxValue* sx_mul(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(a->number * b->number);
    sx_error("Operesheni '*' inahitaji nambari");
    return sx_null();
}

SxValue* sx_div(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER) {
        if (b->number == 0) sx_error("Haiwezekani kugawanya na sifuri");
        return sx_number(a->number / b->number);
    }
    sx_error("Operesheni '/' inahitaji nambari");
    return sx_null();
}

SxValue* sx_mod(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER) {
        if (b->number == 0) sx_error("Haiwezekani kugawanya na sifuri");
        return sx_number((double)((long long)a->number % (long long)b->number));
    }
    sx_error("Operesheni '%' inahitaji nambari");
    return sx_null();
}

SxValue* sx_pow(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_number(pow(a->number, b->number));
    sx_error("Operesheni '**' inahitaji nambari");
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
    sx_error("Ulinganisho '>' unahitaji nambari");
    return sx_null();
}

SxValue* sx_lt(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number < b->number);
    sx_error("Ulinganisho '<' unahitaji nambari");
    return sx_null();
}

SxValue* sx_gte(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number >= b->number);
    sx_error("Ulinganisho '>=' unahitaji nambari");
    return sx_null();
}

SxValue* sx_lte(SxValue *a, SxValue *b) {
    if (a->type == SX_NUMBER && b->type == SX_NUMBER)
        return sx_bool(a->number <= b->number);
    sx_error("Ulinganisho '<=' unahitaji nambari");
    return sx_null();
}

// --- Logical ---

SxValue* sx_not(SxValue *a) {
    return sx_bool(!sx_truthy(a));
}

// --- Print ---

static void sx_print_value(SxValue *v, int quote_strings) {
    if (!v) { printf("tupu"); return; }
    switch (v->type) {
        case SX_NULL: printf("tupu"); break;
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
        case SX_BOOL: printf("%s", v->boolean ? "kweli" : "sikweli"); break;
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
    if (list->type != SX_LIST) sx_error("ongeza() hoja ya kwanza lazima iwe safu");
    if (list->list.len >= list->list.cap) {
        list->list.cap = list->list.cap == 0 ? 4 : list->list.cap * 2;
        list->list.items = (SxValue**)SX_REALLOC(list->list.items, sizeof(SxValue*) * list->list.cap);
    }
    list->list.items[list->list.len++] = item;
}

SxValue* sx_list_get(SxValue *list, SxValue *index) {
    if (list->type != SX_LIST) sx_error("si safu");
    if (index->type != SX_NUMBER) sx_error("Fahirisi lazima iwe nambari");
    int i = (int)index->number;
    if (i < 0 || i >= list->list.len) sx_error("Fahirisi nje ya masafa");
    return list->list.items[i];
}

void sx_list_set(SxValue *list, SxValue *index, SxValue *val) {
    if (list->type != SX_LIST) sx_error("si safu");
    if (index->type != SX_NUMBER) sx_error("Fahirisi lazima iwe nambari");
    int i = (int)index->number;
    if (i < 0 || i >= list->list.len) sx_error("Fahirisi nje ya masafa");
    list->list.items[i] = val;
}

SxValue* sx_list_remove(SxValue *list, SxValue *index) {
    if (list->type != SX_LIST) sx_error("ondoa() hoja ya kwanza lazima iwe safu");
    if (index->type != SX_NUMBER) sx_error("ondoa() hoja ya pili lazima iwe nambari");
    int i = (int)index->number;
    if (i < 0 || i >= list->list.len) sx_error("Fahirisi nje ya masafa");
    SxValue *removed = list->list.items[i];
    for (int j = i; j < list->list.len - 1; j++)
        list->list.items[j] = list->list.items[j + 1];
    list->list.len--;
    return removed;
}

// --- Dict operations (hash table) ---

SxValue* sx_dict_get(SxValue *dict, SxValue *key) {
    if (dict->type != SX_DICT) sx_error("si kamusi");
    if (key->type != SX_STRING) sx_error("Ufunguo wa kamusi lazima uwe tungo");
    SxDictBucket *b = sx_dict_find(&dict->dict, key->string);
    return b ? b->value : sx_null();
}

void sx_dict_set(SxValue *dict, SxValue *key, SxValue *val) {
    if (dict->type != SX_DICT) sx_error("si kamusi");
    if (key->type != SX_STRING) sx_error("Ufunguo wa kamusi lazima uwe tungo");

    // Check if key exists
    SxDictBucket *existing = sx_dict_find(&dict->dict, key->string);
    if (existing) {
        existing->value = val;
        return;
    }

    // Resize if needed
    if ((double)dict->dict.len / dict->dict.capacity >= SX_DICT_LOAD_FACTOR) {
        sx_dict_resize(&dict->dict);
    }

    // Insert into hash table
    unsigned long h = sx_hash(key->string) & (dict->dict.capacity - 1);
    while (dict->dict.buckets[h].used)
        h = (h + 1) & (dict->dict.capacity - 1);

    dict->dict.buckets[h].key = sx_strdup(key->string);
    dict->dict.buckets[h].value = val;
    dict->dict.buckets[h].used = 1;

    // Track insertion order
    sx_dict_add_key(&dict->dict, key->string);
    dict->dict.len++;
}

SxValue* sx_dict_keys(SxValue *dict) {
    if (dict->type != SX_DICT) sx_error("funguo() inahitaji kamusi");
    SxValue *list = sx_list_new();
    for (int i = 0; i < dict->dict.len; i++)
        sx_list_append(list, sx_string(dict->dict.keys[i]));
    return list;
}

SxValue* sx_dict_values(SxValue *dict) {
    if (dict->type != SX_DICT) sx_error("thamani() inahitaji kamusi");
    SxValue *list = sx_list_new();
    for (int i = 0; i < dict->dict.len; i++) {
        SxDictBucket *b = sx_dict_find(&dict->dict, dict->dict.keys[i]);
        if (b) sx_list_append(list, b->value);
    }
    return list;
}

SxValue* sx_dict_has(SxValue *dict, SxValue *key) {
    if (dict->type != SX_DICT) sx_error("ina() hoja ya kwanza lazima iwe kamusi");
    if (key->type != SX_STRING) sx_error("ina() hoja ya pili lazima iwe tungo");
    return sx_bool(sx_dict_find(&dict->dict, key->string) != NULL);
}

// --- Generic index ---

SxValue* sx_index(SxValue *collection, SxValue *idx) {
    switch (collection->type) {
        case SX_LIST: return sx_list_get(collection, idx);
        case SX_DICT: return sx_dict_get(collection, idx);
        case SX_STRING: {
            if (idx->type != SX_NUMBER) sx_error("Fahirisi lazima iwe nambari");
            int i = (int)idx->number;
            if (i < 0 || i >= (int)strlen(collection->string)) sx_error("Fahirisi nje ya masafa");
            char buf[2] = {collection->string[i], '\0'};
            return sx_string(buf);
        }
        default: sx_error("Haiwezi kufikia kwa fahirisi");
    }
    return sx_null();
}

void sx_index_set(SxValue *collection, SxValue *idx, SxValue *val) {
    switch (collection->type) {
        case SX_LIST: sx_list_set(collection, idx, val); break;
        case SX_DICT: sx_dict_set(collection, idx, val); break;
        default: sx_error("Haiwezi kuweka kwa fahirisi");
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
            if (needle->type != SX_STRING) sx_error("ktk kwa tungo inahitaji tungo");
            return sx_bool(strstr(haystack->string, needle->string) != NULL);
        default:
            sx_error("ktk haiwezi kutumika kwa aina hii");
    }
    return sx_bool(0);
}

// --- Utilities ---

SxValue* sx_len(SxValue *v) {
    switch (v->type) {
        case SX_STRING: return sx_number(strlen(v->string));
        case SX_LIST: return sx_number(v->list.len);
        case SX_DICT: return sx_number(v->dict.len);
        default: sx_error("urefu() inahitaji safu, tungo, au kamusi");
    }
    return sx_null();
}

SxValue* sx_type(SxValue *v) {
    switch (v->type) {
        case SX_NUMBER: return sx_string("nambari");
        case SX_STRING: return sx_string("tungo");
        case SX_BOOL: return sx_string("buliani");
        case SX_LIST: return sx_string("safu");
        case SX_DICT: return sx_string("kamusi");
        case SX_NULL: return sx_string("tupu");
        default: return sx_string("haijulikani");
    }
}

SxValue* sx_range(SxValue *start, SxValue *end) {
    if (start->type != SX_NUMBER || end->type != SX_NUMBER)
        sx_error("masafa() inahitaji nambari");
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
            if (end == v->string) sx_error("Haiwezi kubadilisha tungo kuwa nambari");
            return sx_number(n);
        }
        case SX_BOOL: return sx_number(v->boolean ? 1 : 0);
        default: sx_error("Haiwezi kubadilisha kuwa nambari");
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
        case SX_BOOL: return sx_string(v->boolean ? "kweli" : "sikweli");
        case SX_NULL: return sx_string("tupu");
        default: return sx_string("<kitu>");
    }
}

SxValue* sx_to_bool(SxValue *v) {
    return sx_bool(sx_truthy(v));
}

SxValue* sx_concat(int count, ...) {
    va_list args;
    // First pass: calculate total length
    va_start(args, count);
    size_t total = 0;
    for (int i = 0; i < count; i++) {
        SxValue *v = va_arg(args, SxValue*);
        SxValue *s = sx_to_string(v);
        total += strlen(s->string);
    }
    va_end(args);

    // Second pass: build string with running offset (O(n) instead of O(n^2))
    char *result = (char*)SX_MALLOC(total + 1);
    size_t offset = 0;
    va_start(args, count);
    for (int i = 0; i < count; i++) {
        SxValue *v = va_arg(args, SxValue*);
        SxValue *s = sx_to_string(v);
        size_t len = strlen(s->string);
        memcpy(result + offset, s->string, len);
        offset += len;
    }
    result[offset] = '\0';
    va_end(args);

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
            case SX_NUMBER: exp_name = "nambari"; break;
            case SX_STRING: exp_name = "tungo"; break;
            case SX_BOOL: exp_name = "buliani"; break;
            case SX_LIST: exp_name = "safu"; break;
            case SX_DICT: exp_name = "kamusi"; break;
            default: exp_name = "haijulikani"; break;
        }
        snprintf(msg, sizeof(msg), "Aina si sahihi: '%s' ni %s, inahitaji %s", name, got->string, exp_name);
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
