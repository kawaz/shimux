// GhosttyBackend のロジックテスト
// 実際の osascript 実行は行わない（Ghostty が必要なため）。
// ペインID変換や状態管理のテストを行う。

// === pane_id_to_terminal_id ===

///|
test "pane_id_to_terminal_id: removes % prefix" {
  let result = pane_id_to_terminal_id(@core.PaneId("%42"))
  assert_eq(result, "42")
}

///|
test "pane_id_to_terminal_id: no prefix" {
  let result = pane_id_to_terminal_id(@core.PaneId("99"))
  assert_eq(result, "99")
}

///|
test "pane_id_to_terminal_id: empty string" {
  let result = pane_id_to_terminal_id(@core.PaneId(""))
  assert_eq(result, "")
}

///|
test "pane_id_to_terminal_id: % only" {
  let result = pane_id_to_terminal_id(@core.PaneId("%"))
  assert_eq(result, "")
}

// === shell_quote_for_sh ===

///|
test "shell_quote_for_sh: simple string" {
  let result = shell_quote_for_sh("hello")
  assert_eq(result, "'hello'")
}

///|
test "shell_quote_for_sh: with single quote" {
  let result = shell_quote_for_sh("it's")
  assert_eq(result, "'it'\\''s'")
}

///|
test "shell_quote_for_sh: empty string" {
  let result = shell_quote_for_sh("")
  assert_eq(result, "''")
}

// === get_session_state ===

///|
test "get_session_state: returns default state" {
  let backend = GhosttyBackend::new()
  let state = get_session_state(backend)
  assert_eq(state.name, "ghostty")
  assert_eq(state.panes.length(), 0)
}

// === find_active_pane ===

///|
test "find_active_pane: empty panes" {
  let state : @core.SessionState = { name: "test", panes: [], next_pane_id: 0 }
  assert_true(find_active_pane(state) is None)
}

///|
test "find_active_pane: finds active" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId("%1"),
    name: "active",
    cwd: "/",
    width: 80,
    height: 24,
    active: true,
  }
  let state : @core.SessionState = {
    name: "test",
    panes: [pane],
    next_pane_id: 2,
  }
  match find_active_pane(state) {
    Some(p) => assert_eq(p.id, @core.PaneId("%1"))
    None => assert_true(false)
  }
}

// === show_option ===

///|
test "show_option: prefix returns C-b" {
  let backend = GhosttyBackend::new()
  let result = @core.Terminal::show_option(backend, "prefix")
  assert_eq(result, "C-b")
}

///|
test "show_option: unknown returns empty" {
  let backend = GhosttyBackend::new()
  let result = @core.Terminal::show_option(backend, "unknown-option")
  assert_eq(result, "")
}

// === has_session ===

///|
test "has_session: always false (no session management)" {
  let backend = GhosttyBackend::new()
  let result = @core.Terminal::has_session(backend, "any-session")
  assert_eq(result, false)
}

// === display_message ===

///|
test "display_message: session_name" {
  let backend = GhosttyBackend::new()
  let result = @core.Terminal::display_message(backend, "#{session_name}")
  assert_eq(result, "ghostty")
}

// === resolve_nosuspend_path ===

///|
test "resolve_nosuspend_path: normal case with cmd/shimux marker" {
  let result = resolve_nosuspend_path("/path/to/cmd/shimux/shimux")
  assert_eq(result, "/path/to/cmd/nosuspend/nosuspend")
}

///|
test "resolve_nosuspend_path: with .exe suffix" {
  let result = resolve_nosuspend_path("/path/to/cmd/shimux/shimux.exe")
  assert_eq(result, "/path/to/cmd/nosuspend/nosuspend.exe")
}

///|
test "resolve_nosuspend_path: marker not found falls back to nosuspend" {
  let result = resolve_nosuspend_path("some-random-path")
  assert_eq(result, "nosuspend")
}

///|
test "resolve_nosuspend_path: without .exe suffix" {
  let result = resolve_nosuspend_path("/usr/local/bin/cmd/shimux/shimux")
  assert_eq(result, "/usr/local/bin/cmd/nosuspend/nosuspend")
}

///|
test "resolve_nosuspend_path: empty string falls back to nosuspend" {
  let result = resolve_nosuspend_path("")
  assert_eq(result, "nosuspend")
}
