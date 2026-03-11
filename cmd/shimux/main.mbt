///|
/// shimux エントリーポイント
///
/// argv[0] のベースネームが "tmux" → shim モード（tmux パーサ → Terminal trait）
/// それ以外 → wrapper モード（-- 以降を子コマンドとして環境セットアップ + 起動）

// --- FFI バインディング ---

fn to_utf8(s : String) -> Bytes {
  @utf8.encode(s)
}

///|
extern "C" fn shimux_exit(code : Int) -> Unit = "shimux_exit"

///|
#borrow(msg)
extern "C" fn shimux_eprintln(msg : Bytes) -> Unit = "shimux_eprintln"

// --- ヘルパー ---

///|
/// stderr に出力して終了する。
fn die(msg : String) -> Unit {
  shimux_eprintln(to_utf8(msg))
  shimux_exit(1)
}

///|
/// パスからベースネームを取得する。
fn basename(path : String) -> String {
  @wrapper.extract_command_name(path)
}

///|
/// サイズ文字列（"50%", "80" 等）を Int に変換する。
fn parse_size(s : String) -> Int {
  let mut n = 0
  for ch in s {
    if ch >= '0' && ch <= '9' {
      n = n * 10 + ch.to_int() - '0'.to_int()
    } else {
      break
    }
  }
  n
}

///|
/// 文字列中のターゲットを置換する。
fn replace_substring(
  s : String,
  target : String,
  replacement : String,
) -> String {
  let s_chars : Array[Char] = []
  for ch in s {
    s_chars.push(ch)
  }
  let t_chars : Array[Char] = []
  for ch in target {
    t_chars.push(ch)
  }
  let buf = StringBuilder::new()
  let slen = s_chars.length()
  let tlen = t_chars.length()
  let i : Ref[Int] = Ref::new(0)
  while i.val < slen {
    if i.val + tlen <= slen {
      let mut matched = true
      for j in 0..<tlen {
        if s_chars[i.val + j] != t_chars[j] {
          matched = false
          break
        }
      }
      if matched {
        buf.write_string(replacement)
        i.val = i.val + tlen
        continue
      }
    }
    buf.write_char(s_chars[i.val])
    i.val = i.val + 1
  }
  buf.to_string()
}

///|
/// tmux フォーマット文字列中のペイン変数を展開する（PaneId のみ）。
fn format_pane_output(format : String, pane_id : @core.PaneId) -> String {
  replace_substring(format, "#\u007bpane_id}", pane_id.0)
}

///|
/// PaneInfo を使ってフォーマット文字列を展開する。
fn format_pane_info(format : String, pane : @core.PaneInfo) -> String {
  let result = format
  let result = replace_substring(result, "#\u007bpane_id}", pane.id.0)
  let result = replace_substring(
    result,
    "#\u007bpane_width}",
    pane.width.to_string(),
  )
  let result = replace_substring(
    result,
    "#\u007bpane_height}",
    pane.height.to_string(),
  )
  let result = replace_substring(
    result,
    "#\u007bpane_active}",
    if pane.active {
      "1"
    } else {
      "0"
    },
  )
  let result = replace_substring(result, "#\u007bpane_current_path}", pane.cwd)
  let result = replace_substring(result, "#\u007bpane_pid}", "0")
  result
}

///|
/// 親プロセスの全環境変数を収集する。
/// `env` コマンドの出力を解析して "KEY=VALUE" 形式の配列を返す。
/// TMUX_PANE は split 後に設定するため除外する。
fn collect_env_vars() -> Array[String] {
  let output = @ghostty.exec_capture("env")
  let env_vars : Array[String] = []
  let lines = split_by_newline(output)
  for line in lines {
    if line == "" {
      continue
    }
    // TMUX_PANE は除外（split 後にシェルラップ内で設定）
    if starts_with(line, "TMUX_PANE=") {
      continue
    }
    // KEY=VALUE 形式かどうかの簡易チェック（= を含む行）
    if contains_char(line, '=') {
      env_vars.push(line)
    }
  }
  env_vars
}

///|
/// 文字列を改行で分割する。
fn split_by_newline(s : String) -> Array[String] {
  let result : Array[String] = []
  let buf = StringBuilder::new()
  for ch in s {
    if ch == '\n' {
      result.push(buf.to_string())
      buf.reset()
    } else {
      buf.write_char(ch)
    }
  }
  let rest = buf.to_string()
  if rest != "" {
    result.push(rest)
  }
  result
}

///|
/// 文字列が指定プレフィックスで始まるかチェックする。
fn starts_with(s : String, prefix : String) -> Bool {
  let s_chars : Array[Char] = []
  for ch in s {
    s_chars.push(ch)
  }
  let p_chars : Array[Char] = []
  for ch in prefix {
    p_chars.push(ch)
  }
  if s_chars.length() < p_chars.length() {
    return false
  }
  for i in 0..<p_chars.length() {
    if s_chars[i] != p_chars[i] {
      return false
    }
  }
  true
}

///|
/// 文字列に指定文字が含まれるかチェックする。
fn contains_char(s : String, target : Char) -> Bool {
  for ch in s {
    if ch == target {
      return true
    }
  }
  false
}

///|
/// new-session 用のシェルコマンドを組み立てる。
/// cwd が指定されていれば cd してから exec する。
/// /dev/tty にリダイレクトして TTY を確保する。
/// Design rationale: Claude Code は tmux new-session を pipe 付きで spawn するため、
/// 子プロセスの stdin/stdout が TTY でなくなる。本物の tmux と同様に
/// /dev/tty を直接開いて TTY を確保する。
fn build_exec_command(cwd : String, command : Array[String]) -> String {
  let buf = StringBuilder::new()
  if cwd != "" {
    buf.write_string("cd ")
    buf.write_string(@wrapper.shell_quote(cwd))
    buf.write_string(" && ")
  }
  // exec で shell の fd を /dev/tty にリダイレクトし、次の exec でコマンドに引き継ぐ
  buf.write_string("exec </dev/tty >/dev/tty 2>&1; exec")
  for arg in command {
    buf.write_char(' ')
    buf.write_string(@wrapper.shell_quote(arg))
  }
  buf.to_string()
}

// --- メイン ---

///|
fn main {
  let args = @env.args()
  if args.length() == 0 {
    die("shimux: no arguments")
  }
  let exe_name = basename(args[0])
  if exe_name == "tmux" {
    run_shim(args)
  } else {
    run_wrapper(args)
  }
}

// --- Shim モード ---

///|
fn run_shim(args : Array[String]) -> Unit {
  let shim_args : Array[String] = []
  for i in 1..<args.length() {
    shim_args.push(args[i])
  }
  let cmd = @tmux.parse(shim_args) catch {
    @tmux.EmptyInput => {
      println("shimux 0.1.0 (tmux-compatible shim)")
      return
    }
    @tmux.MissingArgument(msg) => {
      die("shimux: missing argument: " + msg)
      return
    }
    @tmux.UnknownCommand(name) => {
      die("shimux: unknown command: " + name)
      return
    }
  }
  let backend = @ghostty.GhosttyBackend::new()
  let terminal = backend.as_terminal()
  dispatch(terminal, cmd)
}

///|
fn dispatch(terminal : &@core.Terminal, cmd : @tmux.TmuxCommand) -> Unit {
  dispatch_inner(terminal, cmd) catch {
    @core.TerminalError(msg) => die("shimux: terminal error: " + msg)
  }
}

///|
fn dispatch_inner(
  terminal : &@core.Terminal,
  cmd : @tmux.TmuxCommand,
) -> Unit raise @core.TerminalError {
  match cmd {
    Version => println("tmux 3.6a")
    HasSession(target~) => {
      let exists = terminal.has_session(target)
      if !exists {
        shimux_exit(1)
        return
      }
    }
    SwitchClient(target~) => {
      // Ghostty にはクライアント概念がないので no-op
      let _ = target
    }
    NewSession(detached~, command~, cwd~, session~, attach_if_exists~) => {
      let _ = attach_if_exists
      let _ = session
      if detached && command.length() > 0 {
        // -d + command: surface configuration 付きで Ghostty split を作成
        // 親プロセスの全環境変数を収集して surface configuration に渡す
        let env_vars = collect_env_vars()
        let _ = terminal.split_pane_with_command(
          @core.Horizontal,
          command,
          cwd,
          env_vars,
        )
        return
      }
      if detached {
        // -d のみ（command なし）: セッション枠だけ作る → no-op
        return
      }
      if command.length() > 0 {
        // -A + command: ターミナルを占有して実行（ブロック）
        let shell_cmd = build_exec_command(cwd, command)
        let exit_code = @wrapper.run_command(shell_cmd)
        shimux_exit(exit_code)
        return
      }
      terminal.new_session("", cwd, "")
    }
    ShowOptions(global~, pane~, value_only~, target~, name~) => {
      let _ = global
      let _ = value_only
      let _ = target
      if pane {
        // ペインレベルオプション: 空文字を返す（Ghostty では未サポート）
        // Claude Code は空応答を正常として処理する
        return
      }
      let value = terminal.show_option(name)
      println(value)
    }
    SetOption(..) =>
      // set-option / set: Ghostty ではスタイル設定は no-op
      // Claude Code のボーダー色、window-active-style 等を受けるが無視する
      ()
    SplitWindow(horizontal~, target~, size~, print_info~, format~) => {
      let _ = target
      let dir = @tmux.split_direction(horizontal)
      let size_int = parse_size(size)
      let pane_id = terminal.split_pane(dir, "", size_int)
      if print_info {
        println(format_pane_output(format, pane_id))
      }
    }
    NewWindow(session~, name~, print_info~, format~) => {
      let _ = session
      let pane_id = terminal.new_window("", name)
      if print_info {
        println(format_pane_output(format, pane_id))
      }
    }
    SendKeys(target~, keys~) => {
      let pane_id = @core.PaneId::PaneId(target)
      for key in keys {
        terminal.send_keys(pane_id, key)
      }
    }
    SelectPane(target~, style~, title~) => {
      let _ = style
      let _ = title
      let pane_id = @core.PaneId::PaneId(target)
      terminal.select_pane(pane_id)
    }
    KillPane(target~) => {
      let pane_id = @core.PaneId::PaneId(target)
      terminal.kill_pane(pane_id)
    }
    ListPanes(format~) => {
      let panes = terminal.list_panes(format)
      for pane in panes {
        println(format_pane_info(format, pane))
      }
    }
    DisplayMessage(print~, format~) => {
      let result = terminal.display_message(format)
      if print {
        println(result)
      }
    }
    Unknown(cmd_name) => die("shimux: unsupported command: " + cmd_name)
  }
}

// --- Wrapper モード ---

///|
fn run_wrapper(args : Array[String]) -> Unit {
  // "--" セパレータを探す
  let mut sep_index = -1
  for i in 1..<args.length() {
    if args[i] == "--" {
      sep_index = i
      break
    }
  }
  if sep_index < 0 || sep_index == args.length() - 1 {
    print_help()
    shimux_exit(if args.length() <= 1 { 1 } else { 0 })
    return
  }
  // "--" 以降をコマンドとして渡す
  let command : Array[String] = []
  for i in (sep_index + 1)..<args.length() {
    command.push(args[i])
  }
  @wrapper.run(command)
}

///|
fn print_help() -> Unit {
  println("shimux - tmux-compatible shim for Ghostty")
  println("")
  println("Usage:")
  println(
    "  shimux -- <command> [args...]    Run command with tmux shim environment",
  )
  println("")
  println(
    "When invoked as 'tmux' (via symlink), acts as a tmux-compatible shim",
  )
  println("that translates tmux commands to Ghostty AppleScript operations.")
}
