//! CLI 引数・環境変数関連の FFI 関数群

use crate::moonbit;
use libc::{c_char, c_int};
use std::ffi::{CStr, CString};
use std::sync::OnceLock;

/// CLI 引数をキャッシュする静的変数
static CACHED_ARGS: OnceLock<Vec<CString>> = OnceLock::new();

/// CLI 引数を初期化・取得するヘルパー
fn cached_args() -> &'static Vec<CString> {
    CACHED_ARGS.get_or_init(|| {
        std::env::args()
            .filter_map(|a| CString::new(a).ok())
            .collect()
    })
}

/// コマンドライン引数の数を返す。
///
/// # 戻り値
/// 引数の数（プログラム名を含む）
#[no_mangle]
pub extern "C" fn shimux_argc() -> c_int {
    cached_args().len() as c_int
}

/// i 番目のコマンドライン引数を MoonBit Bytes として返す。
///
/// # 引数
/// - `i`: 0 始まりのインデックス
///
/// # 戻り値
/// - 有効な範囲: MoonBit Bytes (moonbit_make_bytes_raw で確保)
/// - 範囲外: 空の MoonBit Bytes (長さ 0)
#[no_mangle]
pub extern "C" fn shimux_argv(i: c_int) -> *mut u8 {
    let args = cached_args();
    if i < 0 || (i as usize) >= args.len() {
        return moonbit::str_to_bytes("");
    }
    let arg = &args[i as usize];
    unsafe { moonbit::c_str_to_bytes(arg.as_ptr()) }
}

/// 全コマンドライン引数を改行区切りで結合して MoonBit Bytes として返す。
///
/// Design rationale: shimux_argv(i) を繰り返し呼ぶより、
/// 全引数を1回の呼び出しで取得するほうが効率的。
#[no_mangle]
pub extern "C" fn shimux_args_joined() -> *mut u8 {
    let args: Vec<String> = std::env::args().collect();
    let joined = args.join("\n");
    moonbit::str_to_bytes(&joined)
}

/// 環境変数を取得して MoonBit Bytes として返す。
///
/// # Safety
/// - `name` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 変数が存在する場合: MoonBit Bytes (moonbit_make_bytes_raw で確保)
/// - 変数が存在しない場合またはエラー時: 空の MoonBit Bytes (長さ 0)
#[no_mangle]
pub unsafe extern "C" fn shimux_getenv(name: *const c_char) -> *mut u8 {
    if name.is_null() {
        return moonbit::str_to_bytes("");
    }
    let val = unsafe { libc::getenv(name) };
    unsafe { moonbit::c_str_to_bytes(val) }
}

/// 環境変数を設定する。
///
/// # Safety
/// - `name` と `value` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: 0
/// - エラー時: -1
#[no_mangle]
pub unsafe extern "C" fn shimux_setenv(name: *const c_char, value: *const c_char) -> c_int {
    if name.is_null() || value.is_null() {
        return -1;
    }

    // 名前の検証（空文字列や '=' を含む名前は不正）
    let name_str = match unsafe { CStr::from_ptr(name) }.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };
    if name_str.is_empty() || name_str.contains('=') {
        return -1;
    }

    let ret = unsafe { libc::setenv(name, value, 1) };
    if ret == 0 { 0 } else { -1 }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;
    use std::ptr;

    #[test]
    fn test_argc_positive() {
        // テスト実行時には最低でもプログラム名が入る
        assert!(shimux_argc() >= 1);
    }

    #[test]
    fn test_argv_first() {
        let ptr = shimux_argv(0);
        assert!(!ptr.is_null(), "argv[0] should not be null");
        // テストモックでは null 終端付きなので CStr で読める
        let arg0 = unsafe { CStr::from_ptr(ptr as *const c_char) }
            .to_str()
            .unwrap();
        assert!(!arg0.is_empty(), "argv[0] should not be empty");
    }

    #[test]
    fn test_argv_out_of_range() {
        // 範囲外でも空 Bytes を返す（null ではない）
        let ptr = shimux_argv(-1);
        assert!(!ptr.is_null());
        let s = unsafe { CStr::from_ptr(ptr as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(s, "");

        let ptr = shimux_argv(99999);
        assert!(!ptr.is_null());
        let s = unsafe { CStr::from_ptr(ptr as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(s, "");
    }

    #[test]
    fn test_getenv_setenv() {
        let name = CString::new("SHIMUX_TEST_VAR_ENV").unwrap();
        let value = CString::new("test_value_123").unwrap();

        let ret = unsafe { shimux_setenv(name.as_ptr(), value.as_ptr()) };
        assert_eq!(ret, 0);

        let got = unsafe { shimux_getenv(name.as_ptr()) };
        assert!(!got.is_null());

        let got_str = unsafe { CStr::from_ptr(got as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(got_str, "test_value_123");
    }

    #[test]
    fn test_getenv_nonexistent() {
        let name = CString::new("SHIMUX_NONEXISTENT_VAR_12345").unwrap();
        let got = unsafe { shimux_getenv(name.as_ptr()) };
        // 空 Bytes を返す（null ではない）
        assert!(!got.is_null());
        let got_str = unsafe { CStr::from_ptr(got as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(got_str, "");
    }

    #[test]
    fn test_setenv_null() {
        assert_eq!(unsafe { shimux_setenv(ptr::null(), ptr::null()) }, -1);
    }

    #[test]
    fn test_setenv_invalid_name() {
        let name = CString::new("BAD=NAME").unwrap();
        let value = CString::new("val").unwrap();
        assert_eq!(unsafe { shimux_setenv(name.as_ptr(), value.as_ptr()) }, -1);
    }
}
