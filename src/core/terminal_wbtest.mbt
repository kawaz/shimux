// terminal_wbtest.mbt - Terminal trait のメソッドシグネチャテスト
//
// Terminal は trait なので、テスト用のモック実装を作成して
// 全メソッドが正しいシグネチャで呼び出せることを検証する。

///|
/// テスト用のモック Terminal 実装
struct MockTerminal {
  mut last_op : String
}

///|
fn MockTerminal::new() -> MockTerminal {
  { last_op: "" }
}

///|
/// MockTerminal が Terminal trait を実装
impl Terminal for MockTerminal with has_session(self, session_name) {
  self.last_op = "has_session:" + session_name
  false
}

///|
impl Terminal for MockTerminal with new_session(self, name, cwd, command) {
  self.last_op = "new_session:" + name + "," + cwd + "," + command
}

///|
impl Terminal for MockTerminal with new_window(self, session, name) {
  self.last_op = "new_window:" + session + "," + name
  PaneId("%99")
}

///|
impl Terminal for MockTerminal with split_pane(self, direction, target, size) {
  let dir_str = match direction {
    Horizontal => "h"
    Vertical => "v"
  }
  self.last_op = "split_pane:" + dir_str + "," + target + "," + size.to_string()
  PaneId("%10")
}

///|
impl Terminal for MockTerminal with split_pane_with_command(
  self,
  direction,
  command,
  cwd,
  _env_vars,
) {
  let dir_str = match direction {
    Horizontal => "h"
    Vertical => "v"
  }
  let cmd_str = command
    .iter()
    .fold(init="", fn(acc, s) { if acc == "" { s } else { acc + " " + s } })
  self.last_op = "split_pane_with_command:" +
    dir_str +
    "," +
    cmd_str +
    "," +
    cwd
  PaneId("%20")
}

///|
impl Terminal for MockTerminal with send_keys(self, target, keys) {
  self.last_op = "send_keys:" + target.0 + "," + keys
}

///|
impl Terminal for MockTerminal with send_text(self, target, text) {
  self.last_op = "send_text:" + target.0 + "," + text
}

///|
impl Terminal for MockTerminal with select_pane(self, target) {
  self.last_op = "select_pane:" + target.0
}

///|
impl Terminal for MockTerminal with kill_pane(self, target) {
  self.last_op = "kill_pane:" + target.0
}

///|
impl Terminal for MockTerminal with list_panes(self, format) {
  self.last_op = "list_panes:" + format
  []
}

///|
impl Terminal for MockTerminal with display_message(self, format) {
  self.last_op = "display_message:" + format
  ""
}

///|
impl Terminal for MockTerminal with show_option(self, name) {
  self.last_op = "show_option:" + name
  ""
}

///|
impl Terminal for MockTerminal with resize_pane(self, target, direction, amount) {
  self.last_op = "resize_pane:" +
    target.0 +
    "," +
    direction +
    "," +
    amount.to_string()
}

// --- テストケース ---

///|
/// has_session が呼べること
test "Terminal::has_session" {
  let mock = MockTerminal::new()
  let result = mock.has_session("my-session")
  assert_eq(result, false)
  assert_eq(mock.last_op, "has_session:my-session")
}

///|
/// new_session が呼べること
test "Terminal::new_session" {
  let mock = MockTerminal::new()
  mock.new_session("sess1", "/home", "bash")
  assert_eq(mock.last_op, "new_session:sess1,/home,bash")
}

///|
/// new_window が呼べること
test "Terminal::new_window" {
  let mock = MockTerminal::new()
  let pane_id = mock.new_window("sess1", "editor")
  assert_eq(pane_id, PaneId("%99"))
  assert_eq(mock.last_op, "new_window:sess1,editor")
}

///|
/// split_pane が方向、ターゲット、サイズを取れること
test "Terminal::split_pane" {
  let mock = MockTerminal::new()
  let pane_id = mock.split_pane(Horizontal, "%1", 50)
  assert_eq(pane_id, PaneId("%10"))
  assert_eq(mock.last_op, "split_pane:h,%1,50")
}

///|
/// split_pane_with_command がコマンド・CWD・環境変数を取れること
test "Terminal::split_pane_with_command" {
  let mock = MockTerminal::new()
  let pane_id = mock.split_pane_with_command(
    Horizontal,
    ["claude", "--resume", "abc"],
    "/home/user",
    ["FOO=bar"],
  )
  assert_eq(pane_id, PaneId("%20"))
  assert_eq(
    mock.last_op,
    "split_pane_with_command:h,claude --resume abc,/home/user",
  )
}

///|
/// send_keys が PaneId とキー文字列を取ること
test "Terminal::send_keys" {
  let mock = MockTerminal::new()
  mock.send_keys(PaneId("%1"), "ls -la")
  assert_eq(mock.last_op, "send_keys:%1,ls -la")
}

///|
/// send_text が PaneId とテキストを取ること
test "Terminal::send_text" {
  let mock = MockTerminal::new()
  mock.send_text(PaneId("%2"), "hello world")
  assert_eq(mock.last_op, "send_text:%2,hello world")
}

///|
/// select_pane が PaneId を取ること
test "Terminal::select_pane" {
  let mock = MockTerminal::new()
  mock.select_pane(PaneId("%3"))
  assert_eq(mock.last_op, "select_pane:%3")
}

///|
/// kill_pane が PaneId を取ること
test "Terminal::kill_pane" {
  let mock = MockTerminal::new()
  mock.kill_pane(PaneId("%5"))
  assert_eq(mock.last_op, "kill_pane:%5")
}

///|
/// list_panes がフォーマット文字列を取ること
test "Terminal::list_panes" {
  let mock = MockTerminal::new()
  let panes = mock.list_panes("#{pane_id}")
  assert_eq(panes.length(), 0)
  assert_eq(mock.last_op, "list_panes:#{pane_id}")
}

///|
/// display_message がフォーマット文字列を取り String を返すこと
test "Terminal::display_message" {
  let mock = MockTerminal::new()
  let result = mock.display_message("#{session_name}")
  assert_eq(result, "")
  assert_eq(mock.last_op, "display_message:#{session_name}")
}

///|
/// show_option が名前を取り String を返すこと
test "Terminal::show_option" {
  let mock = MockTerminal::new()
  let result = mock.show_option("default-shell")
  assert_eq(result, "")
  assert_eq(mock.last_op, "show_option:default-shell")
}

///|
/// resize_pane が PaneId, 方向文字列, 量を取ること
test "Terminal::resize_pane" {
  let mock = MockTerminal::new()
  mock.resize_pane(PaneId("%1"), "D", 10)
  assert_eq(mock.last_op, "resize_pane:%1,D,10")
}
