// シグナル番号（macOS / Linux 共通）
///|
pub const SIGHUP : Int = 1

///|
pub const SIGPIPE : Int = 13

///|
pub const SIGTERM : Int = 15

///|
pub const SIGWINCH : Int = 28

///|
extern "C" fn shimux_sig_ignore(signum : Int) -> Int = "shimux_sig_ignore"

///|
extern "C" fn shimux_sig_setup_winch(master_fd : Int) -> Int = "shimux_sig_setup_winch"

///|
pub fn sig_ignore(signum : Int) -> Bool {
  shimux_sig_ignore(signum) == 0
}

///|
pub fn sig_setup_winch(master_fd : Int) -> Bool {
  shimux_sig_setup_winch(master_fd) == 0
}
