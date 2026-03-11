//! shimux FFI: MoonBit native backend から呼び出される C 互換関数群
//!
//! 全関数は `#[no_mangle] pub extern "C" fn shimux_xxx(...)` の形式で公開される。
//! MoonBit 側では `extern "C"` で宣言し、リンク時にこの staticlib を結合する。

pub mod moonbit;
pub mod exec;
pub mod env;
pub mod fs;
pub mod misc;

// 各モジュールの公開関数を re-export（リンカが関数を見つけられるように）
pub use exec::{shimux_exec_capture, shimux_exec_spawn, shimux_free_string, shimux_wait_pid};
pub use env::{shimux_argc, shimux_args_joined, shimux_argv, shimux_getenv, shimux_setenv};
pub use fs::{shimux_mkdir_p, shimux_mkdtemp, shimux_read_file, shimux_symlink, shimux_write_file};
pub use misc::{shimux_eprintln, shimux_exit, shimux_getpid, shimux_self_exe};
