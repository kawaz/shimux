/// Ghostty バックエンド用 FFI バインディング
///
/// Rust FFI (shimux-ffi) の関数を MoonBit native から呼び出すためのバインディング。
/// poc003 のパターンに従い、Bytes (UTF-8) で文字列を受け渡しする。

///|
/// MoonBit String を UTF-8 Bytes に変換する。
/// Design rationale: String.to_bytes() は UTF-16 LE を返すが、
/// Rust FFI は UTF-8 を期待するため @utf8.encode() を使用する。
fn to_utf8(s : String) -> Bytes {
  @utf8.encode(s)
}

/// コマンドを実行して stdout をキャプチャする。

///|
#borrow(cmd)
extern "C" fn shimux_exec_capture(cmd : Bytes) -> Bytes = "shimux_exec_capture"

///|
/// シェルコマンドを実行して標準出力を String で返す。
pub fn exec_capture(cmd : String) -> String {
  let result = shimux_exec_capture(to_utf8(cmd))
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}

/// 環境変数を取得する。

///|
#borrow(name)
extern "C" fn shimux_getenv(name : Bytes) -> Bytes = "shimux_getenv"

///|
/// 環境変数を取得する。存在しない場合は空文字列を返す。
pub fn getenv(name : String) -> String {
  let result = shimux_getenv(to_utf8(name))
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}

/// ディレクトリを再帰的に作成する。

///|
#borrow(path)
extern "C" fn shimux_mkdir_p(path : Bytes) -> Int = "shimux_mkdir_p"

///|
/// ディレクトリを再帰的に作成する。成功時 true。
pub fn mkdir_p(path : String) -> Bool {
  shimux_mkdir_p(to_utf8(path)) == 0
}

/// ファイルに文字列を書き込む。

///|
#borrow(path, content)
extern "C" fn shimux_write_file(path : Bytes, content : Bytes) -> Int = "shimux_write_file"

///|
/// ファイルに文字列を書き込む。成功時 true。
pub fn write_file(path : String, content : String) -> Bool {
  shimux_write_file(to_utf8(path), to_utf8(content)) == 0
}

/// ファイルの内容を読み込む。

///|
#borrow(path)
extern "C" fn shimux_read_file(path : Bytes) -> Bytes = "shimux_read_file"

///|
/// ファイルを読み込む。ファイルが存在しない場合は空文字列を返す。
pub fn read_file(path : String) -> String {
  let result = shimux_read_file(to_utf8(path))
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}

/// stderr にメッセージを出力する。

///|
#borrow(msg)
extern "C" fn shimux_eprintln(msg : Bytes) -> Unit = "shimux_eprintln"

///|
/// stderr にメッセージを出力する（改行付き）。
pub fn eprintln(msg : String) -> Unit {
  shimux_eprintln(to_utf8(msg))
}

/// 現在のプロセスの PID を返す。

///|
extern "C" fn shimux_getpid() -> Int = "shimux_getpid"

///|
/// 現在のプロセスの PID を返す。
pub fn getpid() -> Int {
  shimux_getpid()
}

/// 自分自身の実行ファイルパスを返す（静的、解放不要）。

///|
extern "C" fn shimux_self_exe() -> Bytes = "shimux_self_exe"

///|
/// 自分自身の実行ファイルパスを返す。
pub fn self_exe() -> String {
  let result = shimux_self_exe()
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}
