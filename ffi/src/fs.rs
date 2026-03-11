//! ファイルシステム関連の FFI 関数群

use crate::moonbit;
use libc::{c_char, c_int};
use std::ffi::{CStr, CString};
use std::path::Path;

/// シンボリックリンクを作成する。
///
/// # Safety
/// - `target` と `linkpath` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: 0
/// - エラー時: -1
#[no_mangle]
pub unsafe extern "C" fn shimux_symlink(target: *const c_char, linkpath: *const c_char) -> c_int {
    if target.is_null() || linkpath.is_null() {
        return -1;
    }
    let ret = unsafe { libc::symlink(target, linkpath) };
    if ret == 0 { 0 } else { -1 }
}

/// ディレクトリを再帰的に作成する（`mkdir -p` 相当）。
///
/// # Safety
/// - `path` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時（既存ディレクトリも含む）: 0
/// - エラー時: -1
#[no_mangle]
pub unsafe extern "C" fn shimux_mkdir_p(path: *const c_char) -> c_int {
    if path.is_null() {
        return -1;
    }

    let path_str = match unsafe { CStr::from_ptr(path) }.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    match std::fs::create_dir_all(path_str) {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// 一時ディレクトリを作成してそのパスを MoonBit Bytes として返す。
///
/// `template` はプレフィックスとして使用される（`mkdtemp(3)` の XXXXXX パターンではなく、
/// Rust の `tempdir` 風にプレフィックスとして扱う）。
///
/// # Safety
/// - `template` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: MoonBit Bytes (moonbit_make_bytes_raw で確保)
/// - エラー時: 空の MoonBit Bytes (長さ 0)
#[no_mangle]
pub unsafe extern "C" fn shimux_mkdtemp(template: *const c_char) -> *mut u8 {
    if template.is_null() {
        return moonbit::str_to_bytes("");
    }

    let template_str = match unsafe { CStr::from_ptr(template) }.to_str() {
        Ok(s) => s,
        Err(_) => return moonbit::str_to_bytes(""),
    };

    // template を XXXXXX 付きの mkdtemp(3) 用パターンとして処理
    // template が XXXXXX で終わっていなければ付加する
    let pattern = if template_str.ends_with("XXXXXX") {
        template_str.to_string()
    } else if template_str.ends_with('/') {
        format!("{}tmpXXXXXX", template_str)
    } else {
        format!("{}.XXXXXX", template_str)
    };

    let mut buf = match CString::new(pattern) {
        Ok(cs) => cs.into_bytes_with_nul(),
        Err(_) => return moonbit::str_to_bytes(""),
    };

    let ret = unsafe { libc::mkdtemp(buf.as_mut_ptr() as *mut c_char) };
    if ret.is_null() {
        return moonbit::str_to_bytes("");
    }

    // buf はまだ有効な null 終端バイト列
    unsafe { moonbit::c_str_to_bytes(ret) }
}

/// ファイルに文字列を書き込む。
///
/// ファイルが存在しない場合は新規作成。存在する場合は上書き。
///
/// # Safety
/// - `path` と `content` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: 0
/// - エラー時: -1
#[no_mangle]
pub unsafe extern "C" fn shimux_write_file(path: *const c_char, content: *const c_char) -> c_int {
    if path.is_null() || content.is_null() {
        return -1;
    }

    let path_str = match unsafe { CStr::from_ptr(path) }.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };
    let content_str = match unsafe { CStr::from_ptr(content) }.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    match std::fs::write(path_str, content_str) {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// ファイルの内容を読み込んで MoonBit Bytes として返す。
///
/// # Safety
/// - `path` は有効な null 終端 C 文字列へのポインタでなければならない。
///
/// # 戻り値
/// - 成功時: MoonBit Bytes (moonbit_make_bytes_raw で確保)
/// - エラー時: 空の MoonBit Bytes (長さ 0)
#[no_mangle]
pub unsafe extern "C" fn shimux_read_file(path: *const c_char) -> *mut u8 {
    if path.is_null() {
        return moonbit::str_to_bytes("");
    }

    let path_str = match unsafe { CStr::from_ptr(path) }.to_str() {
        Ok(s) => s,
        Err(_) => return moonbit::str_to_bytes(""),
    };

    if !Path::new(path_str).exists() {
        return moonbit::str_to_bytes("");
    }

    match std::fs::read(path_str) {
        Ok(bytes) => {
            let content = String::from_utf8_lossy(&bytes);
            moonbit::str_to_bytes(&content)
        }
        Err(_) => moonbit::str_to_bytes(""),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;
    use std::ptr;

    #[test]
    fn test_mkdir_p_and_symlink() {
        let dir = std::env::temp_dir().join("shimux_test_mkdir_p/a/b/c");
        let dir_cstr = CString::new(dir.to_str().unwrap()).unwrap();

        // クリーンアップ
        let _ = std::fs::remove_dir_all(std::env::temp_dir().join("shimux_test_mkdir_p"));

        let ret = unsafe { shimux_mkdir_p(dir_cstr.as_ptr()) };
        assert_eq!(ret, 0);
        assert!(dir.is_dir());

        // 既存ディレクトリに対しても成功すること
        let ret = unsafe { shimux_mkdir_p(dir_cstr.as_ptr()) };
        assert_eq!(ret, 0);

        // symlink テスト
        let link = std::env::temp_dir().join("shimux_test_mkdir_p/link_to_c");
        let target_cstr = CString::new(dir.to_str().unwrap()).unwrap();
        let link_cstr = CString::new(link.to_str().unwrap()).unwrap();

        let ret = unsafe { shimux_symlink(target_cstr.as_ptr(), link_cstr.as_ptr()) };
        assert_eq!(ret, 0);
        assert!(link.is_symlink());

        // クリーンアップ
        let _ = std::fs::remove_dir_all(std::env::temp_dir().join("shimux_test_mkdir_p"));
    }

    #[test]
    fn test_mkdir_p_null() {
        assert_eq!(unsafe { shimux_mkdir_p(ptr::null()) }, -1);
    }

    #[test]
    fn test_mkdtemp() {
        let template = CString::new("/tmp/shimux_test_XXXXXX").unwrap();
        let result = unsafe { shimux_mkdtemp(template.as_ptr()) };
        assert!(!result.is_null());

        // テストモックでは null 終端付きなので CStr で読める
        let path = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert!(Path::new(path).is_dir());
        assert!(path.starts_with("/tmp/shimux_test_"));
        assert_ne!(path, "/tmp/shimux_test_XXXXXX");

        // クリーンアップ
        let _ = std::fs::remove_dir(path);
    }

    #[test]
    fn test_mkdtemp_null() {
        // null 入力でも空 Bytes を返す（null ではない）
        let result = unsafe { shimux_mkdtemp(ptr::null()) };
        assert!(!result.is_null());
        let s = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(s, "");
    }

    #[test]
    fn test_write_and_read_file() {
        let path = std::env::temp_dir().join("shimux_test_rw.txt");
        let path_cstr = CString::new(path.to_str().unwrap()).unwrap();
        let content = CString::new("Hello, shimux!").unwrap();

        let ret = unsafe { shimux_write_file(path_cstr.as_ptr(), content.as_ptr()) };
        assert_eq!(ret, 0);

        let result = unsafe { shimux_read_file(path_cstr.as_ptr()) };
        assert!(!result.is_null());

        let read_str = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(read_str, "Hello, shimux!");

        // クリーンアップ
        let _ = std::fs::remove_file(&path);
    }

    #[test]
    fn test_read_file_nonexistent() {
        let path = CString::new("/tmp/shimux_nonexistent_file_12345.txt").unwrap();
        let result = unsafe { shimux_read_file(path.as_ptr()) };
        // 空 Bytes を返す（null ではない）
        assert!(!result.is_null());
        let s = unsafe { CStr::from_ptr(result as *const c_char) }
            .to_str()
            .unwrap();
        assert_eq!(s, "");
    }

    #[test]
    fn test_write_file_null() {
        assert_eq!(unsafe { shimux_write_file(ptr::null(), ptr::null()) }, -1);
    }
}
