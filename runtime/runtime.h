#ifndef SINTAX_RUNTIME_H
#define SINTAX_RUNTIME_H

#include <stdint.h>

// Type tags
typedef enum {
    SX_NULL = 0,
    SX_NUMBER = 1,
    SX_STRING = 2,
    SX_BOOL = 3,
    SX_LIST = 4,
    SX_DICT = 5,
    SX_FUNCTION = 6,
    SX_ERROR = 7
} SxType;

// Forward declaration
typedef struct SxValue SxValue;

// List structure
typedef struct {
    SxValue **items;
    int len;
    int cap;
} SxList;

// Dict bucket (hash table with open addressing)
typedef struct {
    char *key;
    SxValue *value;
    int used;       // 0 = empty, 1 = occupied
} SxDictBucket;

// Dict structure (hash table + insertion-order keys)
typedef struct {
    SxDictBucket *buckets;
    int capacity;          // hash table size (always power of 2)
    int len;               // number of entries
    char **keys;           // insertion-order keys for iteration
    int keys_cap;
} SxDict;

// Function pointer type: takes an array of SxValue* args and count, returns SxValue*
typedef SxValue* (*SxFnPtr)(SxValue** args, int argc);

// Tagged value — the core type of Sintax
struct SxValue {
    SxType type;
    union {
        double number;
        char *string;
        int boolean;
        SxList list;
        SxDict dict;
        SxFnPtr function;  // SX_FUNCTION
    };
};

// Memory helpers (exposed for stdlib modules)
SxValue* sx_alloc(SxType type);
char* sx_strdup(const char *s);

#ifdef SX_USE_GC
#include <gc.h>
#define SX_MALLOC(size) GC_MALLOC(size)
#define SX_REALLOC(ptr, size) GC_REALLOC(ptr, size)
#define SX_FREE(ptr)
#else
#define SX_MALLOC(size) malloc(size)
#define SX_REALLOC(ptr, size) realloc(ptr, size)
#define SX_FREE(ptr) free(ptr)
#endif

// Constructors
SxValue* sx_number(double n);
SxValue* sx_string(const char *s);
SxValue* sx_bool(int b);
SxValue* sx_null(void);
SxValue* sx_list_new(void);
SxValue* sx_dict_new(void);
SxValue* sx_function(SxFnPtr fn);
SxValue* sx_error_new(const char *msg);

// Function calls
SxValue* sx_call(SxValue *fn, SxValue **args, int argc);
SxValue* sx_is_error(SxValue *v);

// Method dispatch
SxValue* sx_method(SxValue *obj, const char *name, SxValue **args, int argc);

// Arithmetic
SxValue* sx_add(SxValue *a, SxValue *b);
SxValue* sx_sub(SxValue *a, SxValue *b);
SxValue* sx_mul(SxValue *a, SxValue *b);
SxValue* sx_div(SxValue *a, SxValue *b);
SxValue* sx_mod(SxValue *a, SxValue *b);
SxValue* sx_pow(SxValue *a, SxValue *b);

// Comparison
SxValue* sx_eq(SxValue *a, SxValue *b);
SxValue* sx_neq(SxValue *a, SxValue *b);
SxValue* sx_gt(SxValue *a, SxValue *b);
SxValue* sx_lt(SxValue *a, SxValue *b);
SxValue* sx_gte(SxValue *a, SxValue *b);
SxValue* sx_lte(SxValue *a, SxValue *b);

// Logical
SxValue* sx_not(SxValue *a);
int sx_truthy(SxValue *a);

// Print
void sx_print(SxValue *v);
void sx_print_multi(int count, ...);

// List operations
void sx_list_append(SxValue *list, SxValue *item);
SxValue* sx_list_get(SxValue *list, SxValue *index);
void sx_list_set(SxValue *list, SxValue *index, SxValue *val);
SxValue* sx_list_remove(SxValue *list, SxValue *index);

// Dict operations
SxDictBucket* sx_dict_find(SxDict *d, const char *key);
SxValue* sx_dict_get(SxValue *dict, SxValue *key);
void sx_dict_set(SxValue *dict, SxValue *key, SxValue *val);
SxValue* sx_dict_keys(SxValue *dict);
SxValue* sx_dict_values(SxValue *dict);
SxValue* sx_dict_has(SxValue *dict, SxValue *key);

// Utilities
SxValue* sx_len(SxValue *v);
SxValue* sx_type(SxValue *v);
SxValue* sx_range(SxValue *start, SxValue *end);
SxValue* sx_to_number(SxValue *v);
SxValue* sx_to_string(SxValue *v);
SxValue* sx_to_bool(SxValue *v);
SxValue* sx_in(SxValue *needle, SxValue *haystack);
SxValue* sx_index(SxValue *collection, SxValue *idx);
void sx_index_set(SxValue *collection, SxValue *idx, SxValue *val);

// String interpolation helper
SxValue* sx_concat(int count, ...);

// Type checking
void sx_check_type(SxValue *v, SxType expected, const char *name);

// Error
void sx_error(const char *msg);

#endif
