#include <errno.h>
#include <unistd.h>
#include <sys/prctl.h>
#include <sys/types.h>
#include <sys/wait.h>
 
int main(int argc, char *argv[]) {
  int ret = -1;
  pid_t pid, forked_pid;
  if (argc == 1)
    return -1;
 
  prctl(PR_SET_CHILD_SUBREAPER, 1);
 
  switch((forked_pid = fork())) {
  case 0:
    return execv(argv[1], &argv[1]);
  case -1:
    return -1;
  default:
    break;
  }
 
  do {
    int wstatus;
    pid = waitpid(-1, &wstatus, 0);
    if (pid == forked_pid)
      ret = WIFEXITED(wstatus) ? WEXITSTATUS(wstatus) : 1;
  } while (pid >= 0 || errno == EINTR);
 
  return ret;
}

