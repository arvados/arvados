#include <signal.h>
#include <dlfcn.h>
#include <stdlib.h>


int sigaction(int signum, const struct sigaction *act, struct sigaction *oldact)
{
  static int (*real_sigaction)(int signum, const struct sigaction *act, struct sigaction *oldact) = NULL;

  // Retrieve the real sigaction we just shadowed.
  if (real_sigaction == NULL) {
    real_sigaction = (void *) dlsym(RTLD_NEXT, "sigaction");
    // Prevent further shadowing in children.
    unsetenv("LD_PRELOAD");
  }

  if (signum == SIGTERM) {
    // Skip this handler, it doesn't do what we want.
    return 0;
  }

  if (signum == SIGHUP) {
    // Install this handler for others as well.
    real_sigaction(SIGTERM, act, oldact);
    real_sigaction(SIGINT, act, oldact);
  }

  // Forward the call the the real sigaction.
  return real_sigaction(signum, act, oldact);
}

// vim: ts=2 sw=2 et
