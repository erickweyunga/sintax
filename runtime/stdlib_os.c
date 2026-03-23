#include "runtime.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>

SxValue* sx_os_read(SxValue *path) {
    FILE *f = fopen(path->string, "r");
    if (!f) { sx_error("Cannot read file"); return sx_null(); }
    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    fseek(f, 0, SEEK_SET);
    char *buf = (char*)SX_MALLOC(len + 1);
    fread(buf, 1, len, f);
    buf[len] = '\0';
    fclose(f);
    SxValue *v = sx_alloc(SX_STRING); v->string = buf; return v;
}

SxValue* sx_os_write(SxValue *path, SxValue *data) {
    FILE *f = fopen(path->string, "w");
    if (!f) { sx_error("Cannot write file"); return sx_null(); }
    SxValue *s = sx_to_string(data);
    fputs(s->string, f);
    fclose(f);
    return sx_null();
}

SxValue* sx_os_exists(SxValue *path) {
    struct stat st;
    return sx_bool(stat(path->string, &st) == 0);
}

SxValue* sx_os_delete(SxValue *path) {
    if (remove(path->string) != 0) sx_error("Cannot delete file");
    return sx_null();
}

SxValue* sx_os_cwd(void) {
    char buf[4096];
    if (getcwd(buf, sizeof(buf)))
        return sx_string(buf);
    return sx_string(".");
}

SxValue* sx_os_getenv(SxValue *name) {
    char *val = getenv(name->string);
    if (!val) return sx_null();
    return sx_string(val);
}

SxValue* sx_os_exec(SxValue *cmd) {
    FILE *f = popen(cmd->string, "r");
    if (!f) return sx_string("");
    char buf[4096];
    int total = 0;
    char *result = (char*)SX_MALLOC(4096);
    result[0] = '\0';
    while (fgets(buf, sizeof(buf), f)) {
        int len = strlen(buf);
        result = (char*)SX_REALLOC(result, total + len + 1);
        memcpy(result + total, buf, len);
        total += len;
    }
    result[total] = '\0';
    // Trim trailing newline
    if (total > 0 && result[total-1] == '\n') result[total-1] = '\0';
    pclose(f);
    SxValue *v = sx_alloc(SX_STRING); v->string = result; return v;
}
