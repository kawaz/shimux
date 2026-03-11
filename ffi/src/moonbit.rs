//! MoonBit Bytes 型との相互変換ヘルパー
//!
//! MoonBit native の `Bytes` は `moonbit_object` ヘッダ（8バイト: rc + meta）付きの
//! managed object。Rust から raw `*const c_char` を返すとヘッダ不正で壊れるため、
//! MoonBit ランタイムの `moonbit_make_bytes_raw` でヘッダ付きバッファを確保し、
//! そこにデータをコピーして返す。

use libc::c_char;

// Production: MoonBit ランタイムが提供するシンボル
#[cfg(not(test))]
extern "C" {
    fn moonbit_make_bytes_raw(size: i32) -> *mut u8;
}

// Test: MoonBit ランタイムが存在しないため malloc で代替
// Design rationale: Rust 単体テストでは MoonBit ヘッダ構造を解釈しないので、
// 単純な malloc で十分。null 終端を追加して CStr::from_ptr でも読めるようにする。
#[cfg(test)]
unsafe fn moonbit_make_bytes_raw(size: i32) -> *mut u8 {
    // size + 1 を確保し、末尾に 0 を置く（テスト時の CStr 互換性のため）
    let alloc_size = if size <= 0 { 1 } else { (size as usize) + 1 };
    let ptr = libc::malloc(alloc_size) as *mut u8;
    if !ptr.is_null() {
        // ゼロ初期化（テスト時の安全策 + null 終端保証）
        std::ptr::write_bytes(ptr, 0, alloc_size);
    }
    ptr
}

/// null 終端 C 文字列を MoonBit Bytes に変換する。
///
/// null の場合は空の Bytes（長さ 0）を返す。
///
/// # Safety
/// - `s` は null または有効な null 終端 C 文字列へのポインタでなければならない。
pub unsafe fn c_str_to_bytes(s: *const c_char) -> *mut u8 {
    if s.is_null() {
        return moonbit_make_bytes_raw(0);
    }
    let len = libc::strlen(s);
    let bytes = moonbit_make_bytes_raw(len as i32);
    if !bytes.is_null() && len > 0 {
        std::ptr::copy_nonoverlapping(s as *const u8, bytes, len);
    }
    bytes
}

/// Rust &str を MoonBit Bytes に変換する。
pub fn str_to_bytes(s: &str) -> *mut u8 {
    let data = s.as_bytes();
    unsafe {
        let bytes = moonbit_make_bytes_raw(data.len() as i32);
        if !bytes.is_null() && !data.is_empty() {
            std::ptr::copy_nonoverlapping(data.as_ptr(), bytes, data.len());
        }
        bytes
    }
}
