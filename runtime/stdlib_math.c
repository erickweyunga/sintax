#include "runtime.h"
#include <math.h>
#include <stdlib.h>

// One-arg
SxValue* sx_math_sqrt(SxValue *v)  { return sx_number(sqrt(v->number)); }
SxValue* sx_math_abs(SxValue *v)   { return sx_number(fabs(v->number)); }
SxValue* sx_math_floor(SxValue *v) { return sx_number(floor(v->number)); }
SxValue* sx_math_ceil(SxValue *v)  { return sx_number(ceil(v->number)); }
SxValue* sx_math_round(SxValue *v) { return sx_number(round(v->number)); }
SxValue* sx_math_sin(SxValue *v)   { return sx_number(sin(v->number)); }
SxValue* sx_math_cos(SxValue *v)   { return sx_number(cos(v->number)); }
SxValue* sx_math_tan(SxValue *v)   { return sx_number(tan(v->number)); }
SxValue* sx_math_asin(SxValue *v)  { return sx_number(asin(v->number)); }
SxValue* sx_math_acos(SxValue *v)  { return sx_number(acos(v->number)); }
SxValue* sx_math_atan(SxValue *v)  { return sx_number(atan(v->number)); }
SxValue* sx_math_log(SxValue *v)   { return sx_number(log(v->number)); }
SxValue* sx_math_log2(SxValue *v)  { return sx_number(log2(v->number)); }
SxValue* sx_math_log10(SxValue *v) { return sx_number(log10(v->number)); }
SxValue* sx_math_exp(SxValue *v)   { return sx_number(exp(v->number)); }
SxValue* sx_math_cbrt(SxValue *v)  { return sx_number(cbrt(v->number)); }
SxValue* sx_math_sign(SxValue *v) {
    if (v->number > 0) return sx_number(1);
    if (v->number < 0) return sx_number(-1);
    return sx_number(0);
}

// No-arg
SxValue* sx_math_pi(void)   { return sx_number(M_PI); }
SxValue* sx_math_e(void)    { return sx_number(M_E); }
SxValue* sx_math_random(void) { return sx_number((double)rand() / RAND_MAX); }

// Two-arg
SxValue* sx_math_pow(SxValue *a, SxValue *b) { return sx_number(pow(a->number, b->number)); }
SxValue* sx_math_min(SxValue *a, SxValue *b) { return sx_number(fmin(a->number, b->number)); }
SxValue* sx_math_max(SxValue *a, SxValue *b) { return sx_number(fmax(a->number, b->number)); }
SxValue* sx_math_random_between(SxValue *a, SxValue *b) {
    return sx_number(a->number + ((double)rand() / RAND_MAX) * (b->number - a->number));
}
