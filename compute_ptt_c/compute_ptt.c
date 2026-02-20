/*
 * compute_ptt.c
 *
 * Reads parquet files from a directory using DuckDB, computes the maximum
 * peak-to-trough ratio across 1-minute and 5-minute backward-looking sliding
 * windows for each msinstanceid, and writes a CSV result.
 *
 * Usage: compute_ptt <parquet_dir> <output.csv> <num_threads> [<n_parquet_files>]
 *
 * n_parquet_files: if > 0, process only the first N part-*.parquet files
 *                  (sorted lexicographically).
 *
 * Build:  gcc -O2 -Wall -pthread -o compute_ptt compute_ptt.c -lduckdb
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <pthread.h>
#include <time.h>
#include <glob.h>
#include "duckdb.h"

/* --------------------------------------------------------------------------
 * Progress bar  (writes to stderr, overwrites the same line)
 * -------------------------------------------------------------------------- */

#define BAR_WIDTH 40

static void progress(uint64_t done, uint64_t total, const char *label)
{
    int filled = (total > 0) ? (int)((double)done / (double)total * BAR_WIDTH) : 0;
    fprintf(stderr, "\r%-18s [", label);
    for (int i = 0; i < BAR_WIDTH; i++)
        fputc(i < filled ? '#' : ' ', stderr);
    fprintf(stderr, "] %llu / %llu",
            (unsigned long long)done, (unsigned long long)total);
    if (done >= total)
        fputc('\n', stderr);
    fflush(stderr);
}

/* --------------------------------------------------------------------------
 * Per-instance storage
 * -------------------------------------------------------------------------- */

typedef struct {
    char     *msid;
    int64_t  *timestamps;
    double   *values;
    uint64_t  n;
    uint64_t  capacity;
    double    max_ptt_1min;
    double    max_ptt_5min;
} Instance;

static void inst_push(Instance *inst, int64_t ts, double val)
{
    if (inst->n >= inst->capacity) {
        inst->capacity = inst->capacity ? inst->capacity * 2 : 64;
        inst->timestamps = realloc(inst->timestamps, inst->capacity * sizeof(int64_t));
        inst->values     = realloc(inst->values,     inst->capacity * sizeof(double));
    }
    inst->timestamps[inst->n] = ts;
    inst->values[inst->n]     = val;
    inst->n++;
}

/* --------------------------------------------------------------------------
 * Dynamic array of instances
 * -------------------------------------------------------------------------- */

typedef struct {
    Instance *data;
    uint64_t  count;
    uint64_t  capacity;
} InstanceVec;

static void ivec_push(InstanceVec *v, Instance inst)
{
    if (v->count >= v->capacity) {
        v->capacity = v->capacity ? v->capacity * 2 : 256;
        v->data = realloc(v->data, v->capacity * sizeof(Instance));
    }
    v->data[v->count++] = inst;
}

/* --------------------------------------------------------------------------
 * Sliding-window max peak-to-trough  (mirrors the Python implementation)
 * -------------------------------------------------------------------------- */

static double sliding_max_ptt(const int64_t *ts, const double *vals,
                               uint64_t n, int64_t window_ms)
{
    double   max_ratio = 1.0;
    uint64_t left = 0;

    for (uint64_t right = 0; right < n; right++) {
        while (ts[right] - ts[left] > window_ms)
            left++;

        double win_min = vals[left];
        double win_max = vals[left];
        for (uint64_t k = left + 1; k <= right; k++) {
            if (vals[k] < win_min) win_min = vals[k];
            if (vals[k] > win_max) win_max = vals[k];
        }
        if (win_min > 0.0) {
            double ratio = win_max / win_min;
            if (ratio > max_ratio) max_ratio = ratio;
        }
    }
    return max_ratio;
}

/* --------------------------------------------------------------------------
 * Worker thread
 * -------------------------------------------------------------------------- */

typedef struct {
    Instance  *instances;
    uint64_t   start;
    uint64_t   end;
    uint64_t  *done; /* shared atomic counter */
} WorkArgs;

static void *worker(void *arg)
{
    WorkArgs *w = (WorkArgs *)arg;
    for (uint64_t i = w->start; i < w->end; i++) {
        Instance *inst = &w->instances[i];
        inst->max_ptt_1min = sliding_max_ptt(inst->timestamps, inst->values,
                                              inst->n, 60000LL);
        inst->max_ptt_5min = sliding_max_ptt(inst->timestamps, inst->values,
                                              inst->n, 300000LL);
        __sync_fetch_and_add(w->done, 1);
    }
    return NULL;
}

/* --------------------------------------------------------------------------
 * Helpers
 * -------------------------------------------------------------------------- */

static void die(const char *msg)
{
    perror(msg);
    exit(1);
}

static int strcmp_compar(const void *a, const void *b)
{
    return strcmp(*(const char *const *)a, *(const char *const *)b);
}

/* Build a DuckDB read_parquet(['f1','f2',...]) file-list string. */
static char *build_filelist(char **paths, int n)
{
    size_t total = 3; /* '[' + ']' + '\0' */
    for (int i = 0; i < n; i++)
        total += strlen(paths[i]) + 4; /* "'...', " */

    char *buf = malloc(total);
    char *p   = buf;
    *p++ = '[';
    for (int i = 0; i < n; i++) {
        if (i > 0) { *p++ = ','; *p++ = ' '; }
        *p++ = '\'';
        size_t len = strlen(paths[i]);
        memcpy(p, paths[i], len);
        p += len;
        *p++ = '\'';
    }
    *p++ = ']';
    *p   = '\0';
    return buf;
}

/* --------------------------------------------------------------------------
 * main
 * -------------------------------------------------------------------------- */

int main(int argc, char **argv)
{
    if (argc < 4 || argc > 6) {
        fprintf(stderr,
            "Usage: %s <parquet_dir> <output.csv> <num_threads>"
            " [<n_parquet_files> [<skip_parquet_files>]]\n",
            argv[0]);
        return 1;
    }

    const char *parquet_dir      = argv[1];
    const char *output_path      = argv[2];
    int         num_threads      = atoi(argv[3]);
    int         n_parquet_files  = argc >= 5 ? atoi(argv[4]) : 0;
    int         skip_parquet_files = argc >= 6 ? atoi(argv[5]) : 0;

    if (num_threads < 1)       num_threads = 1;
    if (skip_parquet_files < 0) skip_parquet_files = 0;

    /* --- Enumerate parquet files --- */
    char pattern[4096];
    snprintf(pattern, sizeof(pattern), "%s/part-*.parquet", parquet_dir);

    glob_t gb;
    if (glob(pattern, 0, NULL, &gb) != 0 || gb.gl_pathc == 0) {
        fprintf(stderr, "No parquet files found matching %s\n", pattern);
        return 1;
    }
    qsort(gb.gl_pathv, gb.gl_pathc, sizeof(char *), strcmp_compar);

    int total_files = (int)gb.gl_pathc;
    int skip = skip_parquet_files < total_files ? skip_parquet_files : total_files;
    int available = total_files - skip;
    int n_files = (n_parquet_files > 0 && n_parquet_files < available)
                  ? n_parquet_files : available;

    fprintf(stderr, "Reading %d parquet file(s) from %s (skipping first %d)\n",
            n_files, parquet_dir, skip);

    char *filelist = build_filelist(gb.gl_pathv + skip, n_files);
    globfree(&gb);

    /* --- Open DuckDB (in-memory) and query --- */
    duckdb_database db;
    duckdb_connection con;
    if (duckdb_open(NULL, &db) == DuckDBError ||
        duckdb_connect(db, &con) == DuckDBError) {
        fprintf(stderr, "Failed to open DuckDB\n");
        return 1;
    }

    /* Pull only the three columns we need, sorted so instances are contiguous. */
    char query[8192];
    snprintf(query, sizeof(query),
        "SELECT msinstanceid,"
               " CAST(\"timestamp\" AS BIGINT),"
               " CAST(memory_utilization AS DOUBLE)"
        " FROM read_parquet(%s)"
        " WHERE memory_utilization IS NOT NULL"
          " AND \"timestamp\" IS NOT NULL"
        " ORDER BY msinstanceid, \"timestamp\"",
        filelist);
    free(filelist);

    duckdb_result result;
    if (duckdb_query(con, query, &result) == DuckDBError) {
        fprintf(stderr, "Query error: %s\n", duckdb_result_error(&result));
        duckdb_destroy_result(&result);
        return 1;
    }
    duckdb_disconnect(&con);
    duckdb_close(&db);

    /* --- Stream sorted rows into per-instance buffers --- */
    InstanceVec instances = {0};
    Instance    cur = {0};

    idx_t nrows = duckdb_row_count(&result);
    fprintf(stderr, "Rows to load: %llu\n", (unsigned long long)nrows);

    uint64_t update_every = nrows / 200 + 1; /* ~0.5% steps */
    for (idx_t row = 0; row < nrows; row++) {
        char   *msid = duckdb_value_varchar(&result, 0, row);
        int64_t ts   = duckdb_value_int64(&result,   1, row);
        double  val  = duckdb_value_double(&result,  2, row);

        if (cur.msid == NULL || strcmp(cur.msid, msid) != 0) {
            if (cur.msid != NULL)
                ivec_push(&instances, cur);
            cur = (Instance){ .msid = strdup(msid) };
        }
        inst_push(&cur, ts, val);
        duckdb_free(msid);

        if (row % update_every == 0 || row + 1 == nrows)
            progress((uint64_t)row + 1, (uint64_t)nrows, "Loading rows");
    }
    if (cur.msid != NULL)
        ivec_push(&instances, cur);

    duckdb_destroy_result(&result);

    fprintf(stderr, "Loaded %llu instances\n",
            (unsigned long long)instances.count);

    /* --- Parallel P2T computation --- */
    if (!instances.count) {
        fprintf(stderr, "No instances found\n");
        return 1;
    }

    if (num_threads > (int)instances.count)
        num_threads = (int)instances.count;

    pthread_t *threads  = malloc(num_threads * sizeof(pthread_t));
    WorkArgs  *workargs = malloc(num_threads * sizeof(WorkArgs));
    if (!threads || !workargs) die("malloc thread arrays");

    uint64_t ptt_done = 0;
    uint64_t chunk = (instances.count + (uint64_t)num_threads - 1)
                     / (uint64_t)num_threads;
    for (int t = 0; t < num_threads; t++) {
        workargs[t].instances = instances.data;
        workargs[t].start     = (uint64_t)t * chunk;
        workargs[t].end       = workargs[t].start + chunk;
        workargs[t].done      = &ptt_done;
        if (workargs[t].end > instances.count)
            workargs[t].end = instances.count;
        pthread_create(&threads[t], NULL, worker, &workargs[t]);
    }

    /* Poll progress while threads run, then join. */
    uint64_t last = (uint64_t)-1;
    while (1) {
        uint64_t d = __sync_fetch_and_add(&ptt_done, 0);
        if (d != last) {
            progress(d, instances.count, "Computing P2T");
            last = d;
        }
        if (d >= instances.count) break;
        struct timespec ts = { 0, 50000000L }; /* 50 ms */
        nanosleep(&ts, NULL);
    }
    for (int t = 0; t < num_threads; t++)
        pthread_join(threads[t], NULL);

    /* --- Write CSV output --- */
    FILE *fout = fopen(output_path, "w");
    if (!fout) die(output_path);

    fprintf(fout, "msinstanceid,max_ptt_1min,max_ptt_5min\n");
    for (uint64_t i = 0; i < instances.count; i++)
        fprintf(fout, "%s,%.9f,%.9f\n",
                instances.data[i].msid,
                instances.data[i].max_ptt_1min,
                instances.data[i].max_ptt_5min);
    fclose(fout);

    /* --- Cleanup --- */
    for (uint64_t i = 0; i < instances.count; i++) {
        free(instances.data[i].msid);
        free(instances.data[i].timestamps);
        free(instances.data[i].values);
    }
    free(instances.data);
    free(threads);
    free(workargs);

    fprintf(stderr, "Wrote %llu results to %s\n",
            (unsigned long long)instances.count, output_path);
    return 0;
}
