///|
extern "C" fn shimux_proc_waitpid(pid : Int, nohang : Int) -> Int64 = "shimux_proc_waitpid"

///|
extern "C" fn shimux_proc_exit(status : Int) = "shimux_proc_exit"

///|
extern "C" fn shimux_proc_kill(pid : Int, signum : Int) -> Int = "shimux_proc_kill"

///|
pub fn proc_waitpid(pid : Int, nohang : Bool) -> (Int, Int)? {
  let result = shimux_proc_waitpid(pid, if nohang { 1 } else { 0 })
  let result_pid = (result & 0xFFFFFFFF_L).to_int()
  if result_pid == -1 { return None }
  let status = (result >> 32).to_int()
  Some((result_pid, status))
}

///|
pub fn proc_exit(status : Int) -> Unit {
  shimux_proc_exit(status)
}

///|
pub fn proc_kill(pid : Int, signum : Int) -> Bool {
  shimux_proc_kill(pid, signum) == 0
}
