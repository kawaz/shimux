//! その他のユーティリティ FFI 関数群

use crate::moonbit;
use libc::{c_char, c_int};
use std::ffi::CStr;

/// 現在のプロセスの PID を返す。
///
/// # 戻り値
/// プロセス ID
#[no_mangle]
pub extern "C" fn shimux_getpid() -> c_int {
    unsafe { libc::getpid() as c_int }
}

/// プロセスの実行ファイルパスを MoonBit Bytes として返す。
///
/// `std::env::current_exe()` の結果を返す。取得できない場合は空の Bytes を返す。
///
/// # 戻り値
/// MoonBit Bytes (moonbit_make_bytes_raw で確保)
#[no_mangle]
pub extern "C" fn shimux_self_exe() -> *mut u8 {
    let path = std::env::current_exe()
        .ok()
        .and_then(|p| p.to_str().map(String::from))
        .unwrap_or_default();
    moonbit::str_to_bytes(&path)
}

/// stderr にメッセージを出力する。
///
/// メッセージの末尾に改行が自動付加される。
///
/// # Safety
/// - `msg` は有効な null 終端 C 文字列へのポインタでなければならない。
///   null の場合は何も出力しない。
#[no_mangle]
pub unsafe extern "C" fn shimux_eprintln(msg: *const c_char) {
    if msg.is_null() {
        return;
    }

    if let Ok(s) = unsafe { CStr::from_ptr(msg) }.to_str() {
        eprintln!("{}", s);
    }
}

/// 指定コードでプロセスを終了する。
///
/// この関数は返らない。
///
/// # 引数
/// - `code`: 終了コード
#[no_mangle]
pub extern "C" fn shimux_exit(code: c_int) -> ! {
    std::process::exit(code as i32)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;
    use std::ptr;

    #[test]
    fn test_getpid() {
        let pid = shimux_getpid();
        assert!(pid > 0, "PID should be positive");
        // std::process::id() と一致すること
        assert_eq!(pid as u32, std::process::id());
    }

    #[test]
    fn test_self_exe() {
        let ptr = shimux_self_exe();
        assert!(!ptr.is_null());

        // テストモックでは null 終端付きなので CStr で読める
        let path = unsafe { CStr::from_ptr(ptr as *const c_char) }
            .to_str()
            .unwrap();
        // テスト実行時はバイナリパスが取得できるはず
        assert!(!path.is_empty(), "self_exe should return non-empty path");
    }

    #[test]
    fn test_self_exe_returns_valid_path() {
        // 毎回新しい MoonBit Bytes を返す（OnceLock キャッシュは廃止）
        let ptr1 = shimux_self_exe();
        let ptr2 = shimux_self_exe();
        // ポインタは異なるが内容は同じ
        let path1 = unsafe { CStr::from_ptr(ptr1 as *const c_char) }
            .to_str()
            .unwrap();
        let path2 = unsafe { CStr::from_ptr(ptr2 as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(path1, path2);
    }

    #[test]
    fn test_eprintln_null() {
        // null を渡しても panic しないこと
        unsafe { shimux_eprintln(ptr::null()) };
    }

    #[test]
    fn test_eprintln_message() {
        let msg = CString::new("test message from shimux_eprintln").unwrap();
        unsafe { shimux_eprintln(msg.as_ptr()) };
        // stderr に出力されること（目視 or キャプチャで確認）
    }

    // shimux_exit のテストは省略（プロセスが終了してしまうため）
}
