//! プロセス実行関連の FFI 関数群

use crate::moonbit;
use libc::{c_char, c_int};
use std::ffi::CStr;
use std::ptr;

/// コマンドを実行して stdout をキャプチャする。
///
/// シェル経由（`/bin/sh -c`）でコマンドを実行し、標準出力を MoonBit Bytes として返す。
///
/// # Safety
/// - `cmd` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: MoonBit Bytes (moonbit_make_bytes_raw で確保)
/// - エラー時: 空の MoonBit Bytes (長さ 0)
#[no_mangle]
pub unsafe extern "C" fn shimux_exec_capture(cmd: *const c_char) -> *mut u8 {
    if cmd.is_null() {
        return moonbit::str_to_bytes("");
    }

    let cmd_cstr = match unsafe { CStr::from_ptr(cmd) }.to_str() {
        Ok(s) => s,
        Err(_) => return moonbit::str_to_bytes(""),
    };

    match std::process::Command::new("/bin/sh")
        .arg("-c")
        .arg(cmd_cstr)
        .output()
    {
        Ok(output) => {
            let stdout = String::from_utf8_lossy(&output.stdout);
            moonbit::str_to_bytes(&stdout)
        }
        Err(_) => moonbit::str_to_bytes(""),
    }
}

/// 互換性のため残す空関数。
///
/// Design rationale: 以前は Rust で alloc した CString の解放用だったが、
/// 現在は全ての文字列が moonbit_make_bytes_raw で確保された MoonBit managed object
/// として返されるため、GC が回収する。既存の MoonBit 側呼び出しコードとの
/// 互換性のために関数自体は残す。
#[no_mangle]
pub unsafe extern "C" fn shimux_free_string(_ptr: *mut c_char) {
    // MoonBit managed bytes は GC で回収されるため何もしない
}

/// コマンドを実行（`posix_spawnp`）して子プロセスの PID を返す。
///
/// # Safety
/// - `argv` は `argc` 個の有効な null 終端 C 文字列ポインタの配列でなければならない。
///
/// # 引数
/// - `argc`: 引数の数
/// - `argv`: 引数文字列の配列
///
/// # 戻り値
/// - 成功時: 子プロセスの PID
/// - エラー時: -1
#[no_mangle]
pub unsafe extern "C" fn shimux_exec_spawn(argc: c_int, argv: *const *const c_char) -> c_int {
    if argc <= 0 || argv.is_null() {
        return -1;
    }

    // argv を Vec<*mut c_char> に変換（NULL 終端付き）
    let argc = argc as usize;
    let mut c_argv: Vec<*mut c_char> = Vec::with_capacity(argc + 1);
    for i in 0..argc {
        let arg = unsafe { *argv.add(i) };
        if arg.is_null() {
            return -1;
        }
        c_argv.push(arg as *mut c_char);
    }
    c_argv.push(ptr::null_mut());

    let mut pid: libc::pid_t = 0;
    let ret = unsafe {
        libc::posix_spawnp(
            &mut pid,
            c_argv[0] as *const c_char,
            ptr::null(),     // file_actions
            ptr::null(),     // attrp
            c_argv.as_ptr(), // argv
            ptr::null(),     // envp (inherit)
        )
    };

    if ret == 0 {
        pid as c_int
    } else {
        -1
    }
}

/// 指定 PID の子プロセスの終了を待つ。
///
/// # 引数
/// - `pid`: 待機する子プロセスの PID
///
/// # 戻り値
/// - 正常終了時: 終了コード
/// - シグナル終了時: 128 + シグナル番号
/// - エラー時: -1
#[no_mangle]
pub extern "C" fn shimux_wait_pid(pid: c_int) -> c_int {
    let mut status: c_int = 0;
    let ret = unsafe { libc::waitpid(pid as libc::pid_t, &mut status, 0) };
    if ret < 0 {
        return -1;
    }

    if libc::WIFEXITED(status) {
        libc::WEXITSTATUS(status)
    } else if libc::WIFSIGNALED(status) {
        128 + libc::WTERMSIG(status)
    } else {
        -1
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;

    #[test]
    fn test_exec_capture_echo() {
        let cmd = CString::new("echo hello").unwrap();
        let result = unsafe { shimux_exec_capture(cmd.as_ptr()) };
        assert!(!result.is_null());

        // テストモックでは null 終端付きで確保されるため CStr で読める
        let output = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(output.trim(), "hello");
    }

    #[test]
    fn test_exec_capture_null() {
        // null 入力でも空 Bytes を返す（null ではない）
        let result = unsafe { shimux_exec_capture(ptr::null()) };
        assert!(!result.is_null());
        let output = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(output, "");
    }

    #[test]
    fn test_exec_capture_multiline() {
        let cmd = CString::new("printf 'line1\\nline2'").unwrap();
        let result = unsafe { shimux_exec_capture(cmd.as_ptr()) };
        assert!(!result.is_null());

        let output = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(output, "line1\nline2");
    }

    #[test]
    fn test_free_string_null() {
        // null ポインタでも安全に呼べること（no-op）
        unsafe { shimux_free_string(ptr::null_mut()) };
    }

    #[test]
    fn test_exec_spawn_and_wait() {
        let arg0 = CString::new("/bin/echo").unwrap();
        let arg1 = CString::new("hello").unwrap();
        let argv: Vec<*const c_char> = vec![arg0.as_ptr(), arg1.as_ptr()];

        let pid = unsafe { shimux_exec_spawn(2, argv.as_ptr()) };
        assert!(pid > 0, "spawn should return a positive PID");

        let status = shimux_wait_pid(pid);
        assert_eq!(status, 0, "echo should exit with status 0");
    }

    #[test]
    fn test_exec_spawn_invalid() {
        let result = unsafe { shimux_exec_spawn(0, ptr::null()) };
        assert_eq!(result, -1);
    }
}
