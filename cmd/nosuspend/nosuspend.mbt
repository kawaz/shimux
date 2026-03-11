/// nosuspend - 子プロセスが suspend されたら即座に SIGCONT する軽量ラッパー
///
/// Ghostty surface configuration の command で直接起動した場合、
/// シェルがいないため Ctrl+Z で suspend すると fg で戻れない。
/// このラッパーは子プロセスの SIGTSTP 停止を検知して即座に再開する。

///|
/// シェル用にシングルクオートでエスケープする。
fn shell_quote(s : String) -> String {
  let buf = StringBuilder::new()
  buf.write_char('\'')
  for ch in s {
    if ch == '\'' {
      buf.write_string("'\\''")
    } else {
      buf.write_char(ch)
    }
  }
  buf.write_char('\'')
  buf.to_string()
}

///|
#borrow(cmd)
extern "C" fn nosuspend_run(cmd : Bytes) -> Int = "nosuspend_run"

///|
pub extern "C" fn exit(code : Int) = "exit"

///|
fn main {
  let args = @env.args()
  if args.length() < 2 {
    println("usage: nosuspend command [args...]")
    exit(1)
  }
  let buf = StringBuilder::new()
  buf.write_string("exec")
  for i in 1..<args.length() {
    buf.write_char(' ')
    buf.write_string(shell_quote(args[i]))
  }
  let exit_code = nosuspend_run(@utf8.encode(buf.to_string()))
  exit(exit_code)
}
