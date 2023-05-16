#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <pthread.h>
#include <stdint.h>

static int verbose = 0;

void usage(char * const*argv) {
	printf("Usage: %s --nthread NTHREAD --niter NITER\n", argv[0]);
	exit(1);
}

void parse_opts(const int argc, char * const*argv, uint64_t *nthread, uint64_t *niter) {
	int c       = 0;
	int opt_idx = 0;

	static struct option long_options[] = {
		/* flags */
		{"verbose",  no_argument, &verbose, 1},
		/* args */
		{"nthread",  required_argument, 0, 't'},
		{"niter",    required_argument, 0, 'i'},
		{0, 0, 0, 0}
  };
	
	while (1) {
		c = getopt_long(argc, argv, "t:i:", long_options, &opt_idx);
		if (c == -1) {
			break;
		}

		switch (c) {
		case 0:
			if (opt_idx != 0) {
				printf("Unexpected flag %s\n", optarg);
				exit(1);
			}
			break;
		case 't':
			*nthread = atoll(optarg);
			break;
		case 'i':
			*niter = atoll(optarg);
			break;
		case '?':
			/* getopt_long already printed error msg */
			usage(argv);
			exit(1);
			break;
		default:
			printf("Unexpected error parsing args\n");
			exit(1);
		}
	}
	if (nthread == 0 || niter == 0) {
		usage(argv);
	}
}

void *spin(void *ptr) {
	uint64_t niter = *((uint64_t *) ptr);
	uint64_t i = 		0;
	uint64_t j = 		0;
	for (i = 0; i < niter; i++) {
		j = j * i + i;
	}
	return NULL;
}

pthread_t *make_pthreads(const uint64_t nthread) {
	pthread_t *ts;
	ts = malloc(sizeof(pthread_t) * nthread);
	return ts;
}

void free_pthreads(pthread_t *ts) {
	free(ts);
}

void create_threads(const uint64_t nthread, uint64_t *niter, pthread_t *ts) {
	int ret = 0;
	uint64_t i   = 0;
	for (i = 0; i < nthread; i++) {
		ret = pthread_create(&ts[i], NULL, spin, niter);
		if (ret != 0) {
			printf("Error pthread create: %d\n", ret);
			exit(1);
		}
	}
}

void join_threads(const uint64_t nthread, const pthread_t *ts) {
	int ret = 0;
	uint64_t i   = 0;
	for (i = 0; i < nthread; i++) {
		ret = pthread_join(ts[i], NULL);
		if (ret != 0) {
			printf("Error pthread join: %d\n", ret);
			exit(1);
		}
	}
}

int main(int argc, char **argv) {
	// Args
	uint64_t nthread = 0;
	uint64_t niter   = 0;

	// Threads
	pthread_t *ts;

	// Other
	int ret = 0;

	parse_opts(argc, argv, &nthread, &niter);
	printf("nthread %llu niter %llu\n", nthread, niter);
	
	ts = make_pthreads(nthread);
	create_threads(nthread, &niter, ts);
	join_threads(nthread, ts);
	free_pthreads(ts);
	
	return 0;
}
