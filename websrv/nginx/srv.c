#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <pthread.h>
#include <fcgi_stdio.h>

static int verbose = 0;

void usage(char * const*argv) {
	fprintf(stderr, "Usage: %s --nworker NWORKER\n", argv[0]);
	exit(1);
}

void parse_opts(const int argc, char * const*argv, int *nworker) {
	int c       = 0;
	int opt_idx = 0;

	static struct option long_options[] = {
		/* flags */
		{"verbose",  no_argument, &verbose, 1},
		/* args */
		{"nworker",  required_argument, 0, 't'},
		{0, 0, 0, 0}
  };
	
	while (1) {
		c = getopt_long(argc, argv, "w:", long_options, &opt_idx);
		if (c == -1) {
			break;
		}

		switch (c) {
		case 0:
			if (opt_idx != 0) {
				fprintf(stderr, "Unexpected flag %s\n", optarg);
				exit(1);
			}
			break;
		case 'w':
			*nworker = atoi(optarg);
			break;
		case '?':
			/* getopt_long already printed error msg */
			usage(argv);
			exit(1);
			break;
		default:
			fprintf(stderr, "Unexpected error parsing args\n");
			exit(1);
		}
	}
	if (*nworker == 0) {
		usage(argv);
	}
}

int spin(int niter) {
	int i = 		0;
	int j = 		0;
	for (i = 0; i < niter; i++) {
		j = j * i + i;
	}
  return j;
}

void *work(void *ptr) {
   while (1) {
    // TODO: get niter from request queue.
  	int niter = 0;//*((int *) ptr);
    spin(niter);
   }
  return NULL;
}

pthread_t *make_pthreads(const int nthread) {
	pthread_t *ts;
	ts = malloc(sizeof(pthread_t) * nthread);
	return ts;
}

void free_pthreads(pthread_t *ts) {
	free(ts);
}


void create_threads(const int nthread, pthread_t *ts) {
	int ret = 0;
	int i   = 0;
	for (i = 0; i < nthread; i++) {
		ret = pthread_create(&ts[i], NULL, work, NULL);
		if (ret != 0) {
			fprintf(stderr, "Error pthread create: %d\n", ret);
			exit(1);
		}
	}
}

void join_threads(const int nthread, const pthread_t *ts) {
	int ret = 0;
	int i   = 0;
	for (i = 0; i < nthread; i++) {
		ret = pthread_join(ts[i], NULL);
		if (ret != 0) {
			fprintf(stderr, "Error pthread join: %d\n", ret);
			exit(1);
		}
	}
}

void serve_requests() {
  fprintf(stderr, "Get ready to server requests\n");
  while (FCGI_Accept() >= 0) {
    // TODO: add to queue, respond.
    printf("Content-type: text/html\r\n"
           "\r\n"
           "<title>Hello world!</title>"
           "<h1>Hello world!</h1>");
  }
}

int main(int argc, char **argv) {
	// Args
	int nworker = 0;

	// Threads
	pthread_t *ts;

	// Other
	int ret = 0;

	parse_opts(argc, argv, &nworker);
	fprintf(stderr, "nworker\n", nworker);
	
	ts = make_pthreads(nworker);
	create_threads(nworker, ts);
  serve_requests();
	join_threads(nworker, ts);
	free_pthreads(ts);
	
	return 0;
}
