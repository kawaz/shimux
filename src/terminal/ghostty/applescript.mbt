/// AppleScript コマンド組み立てロジック
///
/// Ghostty AppleScript API のコマンド文字列を組み立てる。
/// 実際の osascript 実行は run_osascript() を通じて行い、
/// このモジュールは文字列組み立てのみを担当する（テスト容易性のため）。

///|
/// AppleScript 文字列リテラル内のエスケープ処理
/// バックスラッシュとダブルクォートをエスケープする。
fn escape_applescript(s : String) -> String {
  let buf = StringBuilder::new()
  for ch in s {
    if ch == '\\' {
      buf.write_string("\\\\")
    } else if ch == '"' {
      buf.write_string("\\\"")
    } else {
      buf.write_char(ch)
    }
  }
  buf.to_string()
}

///|
/// osascript コマンド文字列を組み立てる。
/// スクリプト内容をシングルクォートで囲んで osascript -e に渡す形式。
fn build_osascript_cmd(script : String) -> String {
  let escaped = escape_single_quote(script)
  let buf = StringBuilder::new()
  buf.write_string("osascript -e '")
  buf.write_string(escaped)
  buf.write_string("'")
  buf.to_string()
}

///|
/// シングルクォート内で使うための文字列エスケープ
/// シングルクォート文字を '\'' に置換する。
fn escape_single_quote(s : String) -> String {
  let buf = StringBuilder::new()
  for ch in s {
    if ch == '\'' {
      buf.write_string("'\\''")
    } else {
      buf.write_char(ch)
    }
  }
  buf.to_string()
}

///|
/// ペイン分割の AppleScript を組み立てる。
/// direction: "right" (horizontal) or "down" (vertical)
fn build_split_script(direction : String) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string(
    "  set term to focused terminal of selected tab of front window\n",
  )
  buf.write_string("  set newTerm to split term direction ")
  buf.write_string(direction)
  buf.write_string("\n")
  buf.write_string("  id of newTerm\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// surface configuration 付きペイン分割の AppleScript を組み立てる。
/// command: シェルコマンド文字列, cwd: 作業ディレクトリ,
/// env_vars: 環境変数の配列 ["KEY=val", ...],
/// wait_after_command: コマンド終了後もペインを閉じない
fn build_split_with_config_script(
  direction : String,
  command : String,
  cwd : String,
  env_vars : Array[String],
  wait_after_command : Bool,
) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  // surface configuration の record を組み立てる
  buf.write_string("  set cfgRecord to {")
  let mut has_prev = false
  if command != "" {
    buf.write_string("command:\"")
    buf.write_string(escape_applescript(command))
    buf.write_string("\"")
    has_prev = true
  }
  if cwd != "" {
    if has_prev {
      buf.write_string(", ")
    }
    buf.write_string("initial working directory:\"")
    buf.write_string(escape_applescript(cwd))
    buf.write_string("\"")
    has_prev = true
  }
  if env_vars.length() > 0 {
    if has_prev {
      buf.write_string(", ")
    }
    buf.write_string("environment variables:{")
    for i, v in env_vars {
      if i > 0 {
        buf.write_string(", ")
      }
      buf.write_string("\"")
      buf.write_string(escape_applescript(v))
      buf.write_string("\"")
    }
    buf.write_string("}")
    has_prev = true
  }
  if wait_after_command {
    if has_prev {
      buf.write_string(", ")
    }
    buf.write_string("wait after command:true")
  }
  buf.write_string("}\n")
  buf.write_string("  set cfg to new surface configuration from cfgRecord\n")
  buf.write_string(
    "  set term to focused terminal of selected tab of front window\n",
  )
  buf.write_string("  set newTerm to split term direction ")
  buf.write_string(direction)
  buf.write_string(" with configuration cfg\n")
  buf.write_string("  id of newTerm\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// テキスト入力（input text）の AppleScript を組み立てる。
/// text が \n で終わる場合、AppleScript の return を結合して
/// テキスト入力と改行を1回の osascript で送信する（タイミング問題の回避）。
fn build_input_text_script(terminal_id : String, text : String) -> String {
  let chars = string_to_chars(text)
  let append_return = chars.length() > 0 && chars[chars.length() - 1] == '\n'
  let text_body = if append_return {
    let buf = StringBuilder::new()
    for i in 0..<(chars.length() - 1) {
      buf.write_char(chars[i])
    }
    buf.to_string()
  } else {
    text
  }
  let escaped_text = escape_applescript(text_body)
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  if append_return {
    buf.write_string("  input text (\"")
    buf.write_string(escaped_text)
    buf.write_string("\" & return) to terminal id \"")
  } else {
    buf.write_string("  input text \"")
    buf.write_string(escaped_text)
    buf.write_string("\" to terminal id \"")
  }
  buf.write_string(terminal_id)
  buf.write_string("\" of selected tab of front window\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// キー送信（send key）の AppleScript を組み立てる。
fn build_send_key_script(
  terminal_id : String,
  key : String,
  modifiers : String,
) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string("  send key \"")
  buf.write_string(key)
  buf.write_string("\"")
  if modifiers != "" {
    buf.write_string(" modifiers \"")
    buf.write_string(modifiers)
    buf.write_string("\"")
  }
  buf.write_string(" to terminal id \"")
  buf.write_string(terminal_id)
  buf.write_string("\" of selected tab of front window\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// フォーカスの AppleScript を組み立てる。
fn build_focus_script(terminal_id : String) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string("  focus terminal id \"")
  buf.write_string(terminal_id)
  buf.write_string("\" of selected tab of front window\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// ペイン閉じの AppleScript を組み立てる。
fn build_close_script(terminal_id : String) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string("  close terminal id \"")
  buf.write_string(terminal_id)
  buf.write_string("\" of selected tab of front window\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// ペイン一覧取得の AppleScript を組み立てる。
fn build_list_panes_script() -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string(
    "  set terms to every terminal of selected tab of front window\n",
  )
  buf.write_string("  set result to \"\"\n")
  buf.write_string("  repeat with t in terms\n")
  buf.write_string(
    "    set result to result & id of t & \",\" & name of t & \",\" & working directory of t & \"\\n\"\n",
  )
  buf.write_string("  end repeat\n")
  buf.write_string("  result\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// リサイズの AppleScript を組み立てる。
fn build_resize_script(
  terminal_id : String,
  direction : String,
  amount : Int,
) -> String {
  let buf = StringBuilder::new()
  buf.write_string("tell application \"Ghostty\"\n")
  buf.write_string("  perform action \"resize_split:")
  buf.write_string(direction)
  buf.write_string(",")
  buf.write_string(amount.to_string())
  buf.write_string("\" on terminal id \"")
  buf.write_string(terminal_id)
  buf.write_string("\" of selected tab of front window\n")
  buf.write_string("end tell")
  buf.to_string()
}

///|
/// tmux のキー名を Ghostty の send key 形式に変換する。
/// 戻り値: (key, modifiers)
fn tmux_keys_to_ghostty(tmux_key : String) -> (String, String) {
  match tmux_key {
    "Enter" => ("return", "")
    "Escape" => ("escape", "")
    "Tab" => ("tab", "")
    "Space" => ("space", "")
    "BSpace" => ("delete", "")
    "Up" => ("up", "")
    "Down" => ("down", "")
    "Left" => ("left", "")
    "Right" => ("right", "")
    _ => {
      // C-x パターン: Ctrl+キー
      // String[n] はバイトアクセス（Char ではない）なので、
      // イテレータで文字を取得する必要がある。
      let chars = string_to_chars(tmux_key)
      if chars.length() == 3 && chars[0] == 'C' && chars[1] == '-' {
        let key_char = chars[2].to_string()
        (key_char, "control")
      } else {
        // そのまま（テキストリテラルとして扱う）
        (tmux_key, "")
      }
    }
  }
}

///|
/// 文字列を Char 配列に変換するヘルパー。
fn string_to_chars(s : String) -> Array[Char] {
  let chars : Array[Char] = []
  for ch in s {
    chars.push(ch)
  }
  chars
}

///|
/// list-panes の AppleScript 出力をパースして PaneInfo 配列にする。
/// 出力形式: "id,name,cwd\n" が繰り返される。
fn parse_list_panes_output(output : String) -> Array[@core.PaneInfo] {
  let panes : Array[@core.PaneInfo] = []
  let lines = split_lines(output)
  for line in lines {
    if line == "" {
      continue
    }
    let parts = split_by_comma(line)
    if parts.length() >= 3 {
      let id_str = parts[0]
      let name = parts[1]
      let cwd = parts[2]
      let buf = StringBuilder::new()
      buf.write_string("%")
      buf.write_string(id_str)
      panes.push({
        id: @core.PaneId(buf.to_string()),
        name,
        cwd,
        width: 0,
        height: 0,
        active: false,
      })
    }
  }
  panes
}

///|
/// 文字列を改行で分割する。
fn split_lines(s : String) -> Array[String] {
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
/// 文字列をカンマで分割する。
fn split_by_comma(s : String) -> Array[String] {
  let result : Array[String] = []
  let buf = StringBuilder::new()
  for ch in s {
    if ch == ',' {
      result.push(buf.to_string())
      buf.reset()
    } else {
      buf.write_char(ch)
    }
  }
  result.push(buf.to_string())
  result
}

///|
/// tmux フォーマット文字列の変数を展開する。
/// #{session_name}, #{pane_id}, #{pane_width} 等を対応する値に置換する。
fn format_var_expand(
  format : String,
  state : @core.SessionState,
  pane : @core.PaneInfo?,
) -> String {
  let buf = StringBuilder::new()
  let chars : Array[Char] = []
  for ch in format {
    chars.push(ch)
  }
  let len = chars.length()
  let i : Ref[Int] = Ref::new(0)
  while i.val < len {
    if i.val + 1 < len && chars[i.val] == '#' && chars[i.val + 1] == '{' {
      // #{...} パターンを探す
      let start = i.val + 2
      let mut end = start
      while end < len && chars[end] != '}' {
        end = end + 1
      }
      if end < len {
        // 変数名を抽出
        let var_buf = StringBuilder::new()
        let mut j = start
        while j < end {
          var_buf.write_char(chars[j])
          j = j + 1
        }
        let var_name = var_buf.to_string()
        let value = resolve_format_var(var_name, state, pane)
        buf.write_string(value)
        i.val = end + 1
      } else {
        buf.write_char(chars[i.val])
        i.val = i.val + 1
      }
    } else {
      buf.write_char(chars[i.val])
      i.val = i.val + 1
    }
  }
  buf.to_string()
}

///|
/// フォーマット変数名を値に解決する。
fn resolve_format_var(
  name : String,
  state : @core.SessionState,
  pane : @core.PaneInfo?,
) -> String {
  match name {
    "session_name" => state.name
    "window_index" => "0"
    "pane_id" =>
      match pane {
        Some(p) => p.id.0
        None => "%0"
      }
    "pane_pid" => "0"
    "pane_current_path" =>
      match pane {
        Some(p) => p.cwd
        None => ""
      }
    "pane_width" =>
      match pane {
        Some(p) => p.width.to_string()
        None => "80"
      }
    "pane_height" =>
      match pane {
        Some(p) => p.height.to_string()
        None => "24"
      }
    "pane_active" =>
      match pane {
        Some(p) => if p.active { "1" } else { "0" }
        None => "0"
      }
    _ => ""
  }
}

///|
/// osascript を実行して結果を返す。
/// 空の結果は osascript 実行の失敗を示す可能性があるが、
/// 正常な空出力（void 戻り値の AppleScript）もあるためエラーにしない。
/// 空スクリプト渡し時には TerminalError を raise する。
pub fn run_osascript(script : String) -> String raise @core.TerminalError {
  if script == "" {
    raise @core.TerminalError("empty AppleScript")
  }
  let cmd = build_osascript_cmd(script)
  let result = exec_capture(cmd)
  // osascript の出力は末尾に改行が付くことがあるので trim
  trim_trailing_newline(result)
}

///|
/// 末尾の改行文字を除去する。
fn trim_trailing_newline(s : String) -> String {
  let chars : Array[Char] = []
  for ch in s {
    chars.push(ch)
  }
  let mut end = chars.length()
  while end > 0 && (chars[end - 1] == '\n' || chars[end - 1] == '\r') {
    end = end - 1
  }
  let buf = StringBuilder::new()
  let mut i = 0
  while i < end {
    buf.write_char(chars[i])
    i = i + 1
  }
  buf.to_string()
}
