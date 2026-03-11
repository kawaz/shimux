/// Wrapper モードで使用する FFI 関数群
///
/// Rust FFI (ffi/) の C ABI 関数への MoonBit バインディング。
/// 文字列は UTF-8 エンコードの Bytes として受け渡す。

///|
/// MoonBit String を UTF-8 エンコードの Bytes に変換する。
///
/// Design rationale: String.to_bytes() は UTF-16 LE を生成するが、
/// Rust FFI は UTF-8 を期待する。@utf8.encode() で正しい UTF-8 Bytes を生成する。
fn to_utf8(s : String) -> Bytes {
  @utf8.encode(s)
}

///|
/// UTF-8 Bytes を MoonBit String にデコードする。
/// 空の Bytes（null ポインタ相当）の場合は空文字列を返す。
fn from_utf8(b : Bytes) -> String {
  if b.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(b[:])
  }
}

// --- extern "C" 宣言: Rust FFI ---

///|
/// 一時ディレクトリを作成してパスを返す。
/// 戻り値は shimux_free_string で解放が必要。
#borrow(template)
extern "C" fn ffi_mkdtemp(template : Bytes) -> Bytes = "shimux_mkdtemp"

///|
/// 確保された文字列を解放する。
#borrow(ptr)
extern "C" fn ffi_free_string(ptr : Bytes) -> Unit = "shimux_free_string"

///|
/// 自分自身の実行ファイルパスを返す（静的、解放不要）。
extern "C" fn ffi_self_exe() -> Bytes = "shimux_self_exe"

///|
/// 現在のプロセスの PID を返す。
extern "C" fn ffi_getpid() -> Int = "shimux_getpid"

///|
/// 環境変数を取得する。
#borrow(name)
extern "C" fn ffi_getenv(name : Bytes) -> Bytes = "shimux_getenv"

///|
/// 環境変数を設定する。成功時 0、エラー時 -1。
#borrow(name, value)
extern "C" fn ffi_setenv(name : Bytes, value : Bytes) -> Int = "shimux_setenv"

///|
/// シンボリックリンクを作成する。成功時 0、エラー時 -1。
#borrow(target, linkpath)
extern "C" fn ffi_symlink(target : Bytes, linkpath : Bytes) -> Int = "shimux_symlink"

///|
/// ディレクトリを再帰的に作成する。成功時 0、エラー時 -1。
#borrow(path)
extern "C" fn ffi_mkdir_p(path : Bytes) -> Int = "shimux_mkdir_p"

///|
/// stderr にメッセージを出力する。
#borrow(msg)
extern "C" fn ffi_eprintln(msg : Bytes) -> Unit = "shimux_eprintln"

///|
/// 指定コードでプロセスを終了する（返らない）。
extern "C" fn ffi_exit(code : Int) -> Unit = "shimux_exit"

///|
/// 子プロセスの終了を待つ。終了コードを返す。エラー時 -1。
extern "C" fn ffi_wait_pid(pid : Int) -> Int = "shimux_wait_pid"

// --- extern "C" 宣言: libc 直接呼び出し ---

///|
/// libc の system(3) を呼び出す。
/// 子プロセスをシェル経由で起動し、waitpid(2) の status を返す。
#borrow(cmd)
extern "C" fn c_system(cmd : Bytes) -> Int = "system"

// --- 高レベルラッパー ---

///|
/// 一時ディレクトリを作成してパスを返す。
pub fn mkdtemp(template : String) -> String {
  let result = ffi_mkdtemp(to_utf8(template))
  if result.length() == 0 {
    ""
  } else {
    let s = from_utf8(result)
    ffi_free_string(result)
    s
  }
}

///|
/// 自分自身の実行ファイルパスを返す。
pub fn self_exe() -> String {
  from_utf8(ffi_self_exe())
}

///|
/// 現在のプロセスの PID を返す。
pub fn getpid() -> Int {
  ffi_getpid()
}

///|
/// 環境変数を取得する。未設定時は空文字列。
pub fn getenv(name : String) -> String {
  from_utf8(ffi_getenv(to_utf8(name)))
}

///|
/// 環境変数を設定する。成功時 true。
pub fn setenv(name : String, value : String) -> Bool {
  ffi_setenv(to_utf8(name), to_utf8(value)) == 0
}

///|
/// シンボリックリンクを作成する。成功時 true。
pub fn symlink(target : String, linkpath : String) -> Bool {
  ffi_symlink(to_utf8(target), to_utf8(linkpath)) == 0
}

///|
/// ディレクトリを再帰的に作成する。成功時 true。
pub fn mkdir_p(path : String) -> Bool {
  ffi_mkdir_p(to_utf8(path)) == 0
}

///|
/// stderr にメッセージを出力する。
pub fn eprintln(msg : String) -> Unit {
  ffi_eprintln(to_utf8(msg))
}

///|
/// 指定コードでプロセスを終了する（返らない）。
pub fn exit(code : Int) -> Unit {
  ffi_exit(code)
}

///|
/// 子プロセスの終了を待つ。終了コードを返す。エラー時 -1。
pub fn wait_pid(pid : Int) -> Int {
  ffi_wait_pid(pid)
}

///|
/// コマンドをシェル経由で実行し、子プロセスの終了コードを返す。
///
/// libc の system(3) を使い、stdin/stdout/stderr は引き継ぐ。
/// system(3) の戻り値は waitpid(2) の status なので、
/// WIFEXITED なら WEXITSTATUS を返し、それ以外は 128+signal を返す。
pub fn run_command(cmd : String) -> Int {
  let status = c_system(to_utf8(cmd))
  if status == -1 {
    // system() 自体のエラー
    -1
  } else {
    // waitpid の status をデコード
    // WIFEXITED: (status & 0x7f) == 0
    // WEXITSTATUS: (status >> 8) & 0xff
    // WIFSIGNALED: ((status & 0x7f) + 1) >> 1 > 0
    // WTERMSIG: status & 0x7f
    let termsig = status & 0x7f
    if termsig == 0 {
      // 正常終了: WEXITSTATUS
      (status >> 8) & 0xff
    } else {
      // シグナル終了
      128 + termsig
    }
  }
}
