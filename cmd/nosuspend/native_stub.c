#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/wait.h>
#include <unistd.h>
#include <spawn.h>
#include "moonbit.h"

extern char **environ;

static pid_t ns_child;
static void ns_forward(int sig) { if (ns_child > 0) kill(ns_child, sig); }

/* posix_spawn + signal forwarding + SIGTSTP auto-resume.
   cmd is executed via /bin/sh -c. Returns child exit code. */
MOONBIT_FFI_EXPORT
int32_t nosuspend_run(moonbit_bytes_t cmd) {
    const char *argv[] = {"/bin/sh", "-c", (const char *)cmd, NULL};

    posix_spawnattr_t attr;
    posix_spawnattr_init(&attr);
    /* Reset all signals to default in the spawned child */
    sigset_t all_sigs;
    sigfillset(&all_sigs);
    posix_spawnattr_setsigdefault(&attr, &all_sigs);
    posix_spawnattr_setflags(&attr, POSIX_SPAWN_SETSIGDEF);

    int err = posix_spawn(&ns_child, "/bin/sh", NULL, &attr,
                          (char *const *)argv, environ);
    posix_spawnattr_destroy(&attr);
    if (err != 0) return 1;

    signal(SIGTSTP, SIG_IGN);
    signal(SIGINT,  ns_forward);
    signal(SIGTERM, ns_forward);
    signal(SIGQUIT, ns_forward);
    signal(SIGHUP,  ns_forward);

    int st;
    for (;;) {
        if (waitpid(ns_child, &st, WUNTRACED) < 0) return 1;
        if (WIFSTOPPED(st))  { kill(ns_child, SIGCONT); continue; }
        if (WIFEXITED(st))   return WEXITSTATUS(st);
        if (WIFSIGNALED(st)) return 128 + WTERMSIG(st);
    }
}
