/// GhosttyBackend の Terminal trait 実装
///
/// Ghostty AppleScript API を通じて各ターミナル操作を実行する。
/// AppleScript の組み立ては applescript.mbt に委譲し、
/// 実行は run_osascript() を通じて行う。

///|
/// セッションが存在するか確認する。
/// shimux では Ghostty ウィンドウの存在を確認する。
/// 現状は常に true を返す（shimux 管理下でのみ呼ばれるため）。
impl @core.Terminal for GhosttyBackend with has_session(self, name) {
  // shimux はセッション管理を持たないため常に false を返す。
  // Claude Code はこれを受けて new-session でセッションを作成する。
  let _ = get_session_state(self)
  let _ = name
  false
}

///|
/// セッションを新規作成する。
/// Ghostty では no-op（shimux wrapper が既にセッション概念を管理）。
impl @core.Terminal for GhosttyBackend with new_session(
  self,
  _name,
  _cwd,
  _command,
) {
  let _ = self.app_name
}

///|
/// 新しいウィンドウを作成する。
/// Ghostty では split で代替するため、split と同様に処理する。
impl @core.Terminal for GhosttyBackend with new_window(self, _session, _name) {
  let _ = self.app_name
  let script = build_split_script("right")
  let result = run_osascript(script)
  let buf = StringBuilder::new()
  buf.write_string("%")
  buf.write_string(trim_trailing_newline(result))
  @core.PaneId(buf.to_string())
}

///|
/// ペインを分割する。
/// SplitDirection に応じて Ghostty の split コマンドを実行する。
impl @core.Terminal for GhosttyBackend with split_pane(
  self,
  direction,
  _target,
  _size,
) {
  let _ = self.app_name
  let dir = match direction {
    @core.Horizontal => "right"
    @core.Vertical => "down"
  }
  let script = build_split_script(dir)
  let result = run_osascript(script)
  let buf = StringBuilder::new()
  buf.write_string("%")
  buf.write_string(trim_trailing_newline(result))
  @core.PaneId(buf.to_string())
}

///|
/// ペインを分割してコマンドを実行する。
/// surface configuration を使い、コマンド・CWD・環境変数を直接指定する。
/// TMUX_PANE は split 前に ID が不明なため、command のシェルラップ内で設定する。
impl @core.Terminal for GhosttyBackend with split_pane_with_command(
  self,
  direction,
  command,
  cwd,
  env_vars,
) {
  let _ = self.app_name
  let dir = match direction {
    @core.Horizontal => "right"
    @core.Vertical => "down"
  }
  // nosuspend でラップ: Ctrl+Z で suspend されたら即 SIGCONT で復帰させる
  // shimux 自身のパスから nosuspend のパスを推定
  let self_exe = self_exe()
  let nosuspend_path = resolve_nosuspend_path(self_exe)
  // sh -c で TMUX_PANE を設定してから exec nosuspend command...
  let cmd_buf = StringBuilder::new()
  cmd_buf.write_string("export TMUX_PANE=%$GHOSTTY_SURFACE_ID && exec ")
  cmd_buf.write_string(shell_quote_for_sh(nosuspend_path))
  for arg in command {
    cmd_buf.write_char(' ')
    cmd_buf.write_string(shell_quote_for_sh(arg))
  }
  let shell_cmd = cmd_buf.to_string()
  let wrapped_cmd = "/bin/sh -c " + shell_quote_for_sh(shell_cmd)
  let script = build_split_with_config_script(
    dir, wrapped_cmd, cwd, env_vars, true,
  )
  let result = run_osascript(script)
  let buf = StringBuilder::new()
  buf.write_string("%")
  buf.write_string(trim_trailing_newline(result))
  @core.PaneId(buf.to_string())
}

///|
/// shimux のパスから nosuspend バイナリのパスを推定する。
/// .../cmd/shimux/shimux.exe → .../cmd/nosuspend/nosuspend.exe
fn resolve_nosuspend_path(self_exe : String) -> String {
  let marker = "cmd/shimux/shimux"
  let replacement = "cmd/nosuspend/nosuspend"
  let marker_chars = string_to_chars(marker)
  let exe_chars = string_to_chars(self_exe)
  let mut found = -1
  for i in 0..<exe_chars.length() {
    if i + marker_chars.length() <= exe_chars.length() {
      let mut matched = true
      for j in 0..<marker_chars.length() {
        if exe_chars[i + j] != marker_chars[j] {
          matched = false
          break
        }
      }
      if matched {
        found = i
        break
      }
    }
  }
  if found >= 0 {
    let buf = StringBuilder::new()
    for i in 0..<found {
      buf.write_char(exe_chars[i])
    }
    buf.write_string(replacement)
    // .exe サフィックスがあれば付与
    let remaining_start = found + marker_chars.length()
    for i in remaining_start..<exe_chars.length() {
      buf.write_char(exe_chars[i])
    }
    buf.to_string()
  } else {
    "nosuspend"
  }
}

///|
/// シェル用にシングルクオートでエスケープする（backend 内部用）。
fn shell_quote_for_sh(s : String) -> String {
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
/// 指定ペインにキーを送信する。
/// tmux のキー名を Ghostty 形式に変換して送信する。
impl @core.Terminal for GhosttyBackend with send_keys(self, target, keys) {
  let _ = self.app_name
  let terminal_id = pane_id_to_terminal_id(target)
  let (key, modifiers) = tmux_keys_to_ghostty(keys)
  if key == keys && modifiers == "" {
    // 変換されなかった（特殊キーでない）: テキストとして送信
    let script = build_input_text_script(terminal_id, keys)
    let _ = run_osascript(script)
  } else {
    // 特殊キーまたは修飾キー付き: キーイベントとして送信
    let script = build_send_key_script(terminal_id, key, modifiers)
    let _ = run_osascript(script)
  }
}

///|
/// 指定ペインにテキストを送信する（input text 用）。
impl @core.Terminal for GhosttyBackend with send_text(self, target, text) {
  let _ = self.app_name
  let terminal_id = pane_id_to_terminal_id(target)
  let script = build_input_text_script(terminal_id, text)
  let _ = run_osascript(script)
}

///|
/// 指定ペインを選択（フォーカス）する。
impl @core.Terminal for GhosttyBackend with select_pane(self, target) {
  let _ = self.app_name
  let terminal_id = pane_id_to_terminal_id(target)
  let script = build_focus_script(terminal_id)
  let _ = run_osascript(script)
}

///|
/// 指定ペインを削除する。
impl @core.Terminal for GhosttyBackend with kill_pane(self, target) {
  let _ = self.app_name
  let terminal_id = pane_id_to_terminal_id(target)
  let script = build_close_script(terminal_id)
  let _ = run_osascript(script)
}

///|
/// ペイン一覧を取得する。
impl @core.Terminal for GhosttyBackend with list_panes(self, _format) {
  let _ = self.app_name
  let script = build_list_panes_script()
  let result = run_osascript(script)
  parse_list_panes_output(result)
}

///|
/// フォーマット変数を展開してメッセージを返す。
impl @core.Terminal for GhosttyBackend with display_message(self, format) {
  let _ = self.app_name
  let state = get_session_state(self)
  let active_pane = find_active_pane(state)
  format_var_expand(format, state, active_pane)
}

///|
/// オプション値を取得する。
/// "prefix" の場合は "C-b" を返す（固定値）。
impl @core.Terminal for GhosttyBackend with show_option(self, name) {
  let _ = self.app_name
  match name {
    "prefix" => "C-b"
    _ => ""
  }
}

///|
/// ペインをリサイズする。
impl @core.Terminal for GhosttyBackend with resize_pane(
  self,
  target,
  direction,
  amount,
) {
  let _ = self.app_name
  let terminal_id = pane_id_to_terminal_id(target)
  let ghostty_dir = match direction {
    "U" => "up"
    "D" => "down"
    "L" => "left"
    "R" => "right"
    _ => direction
  }
  let script = build_resize_script(terminal_id, ghostty_dir, amount)
  let _ = run_osascript(script)
}

///|
/// PaneId ("%42") から terminal ID 文字列 ("42") を取り出す。
fn pane_id_to_terminal_id(pane_id : @core.PaneId) -> String {
  let s = pane_id.0
  let chars = string_to_chars(s)
  if chars.length() > 0 && chars[0] == '%' {
    let buf = StringBuilder::new()
    let mut i = 1
    while i < chars.length() {
      buf.write_char(chars[i])
      i = i + 1
    }
    buf.to_string()
  } else {
    s
  }
}

///|
/// 現在のセッション状態を取得する。
/// 初期段階ではシンプルな固定値を返す。
fn get_session_state(_self : GhosttyBackend) -> @core.SessionState {
  { name: "ghostty", panes: [], next_pane_id: 0 }
}

///|
/// アクティブなペインを探す。
fn find_active_pane(state : @core.SessionState) -> @core.PaneInfo? {
  for pane in state.panes {
    if pane.active {
      return Some(pane)
    }
  }
  None
}
