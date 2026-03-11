// types_wbtest.mbt - PaneInfo, FormatVar, SessionState の型テスト

///|
/// PaneInfo の新フィールド（width, height, active）が構築・参照できること
test "PaneInfo has width, height, active fields" {
  let info : PaneInfo = {
    id: PaneId("%1"),
    name: "test-pane",
    cwd: "/home/user",
    width: 80,
    height: 24,
    active: true,
  }
  assert_eq(info.id, PaneId("%1"))
  assert_eq(info.name, "test-pane")
  assert_eq(info.cwd, "/home/user")
  assert_eq(info.width, 80)
  assert_eq(info.height, 24)
  assert_eq(info.active, true)
}

///|
/// PaneInfo: active が false のケース
test "PaneInfo inactive pane" {
  let info : PaneInfo = {
    id: PaneId("%2"),
    name: "bg-pane",
    cwd: "/tmp",
    width: 120,
    height: 40,
    active: false,
  }
  assert_eq(info.active, false)
  assert_eq(info.width, 120)
  assert_eq(info.height, 40)
}

///|
/// FormatVar の各バリアントが構築できること
test "FormatVar variants exist" {
  let vars : Array[FormatVar] = [
    SessionName,
    WindowIndex,
    PaneId,
    PanePid,
    PaneCurrentPath,
    PaneWidth,
    PaneHeight,
    PaneActive,
  ]
  assert_eq(vars.length(), 8)
}

///|
/// SessionState の構築と参照
test "SessionState construction" {
  let pane1 : PaneInfo = {
    id: PaneId("%1"),
    name: "main",
    cwd: "/home/user",
    width: 80,
    height: 24,
    active: true,
  }
  let pane2 : PaneInfo = {
    id: PaneId("%2"),
    name: "sub",
    cwd: "/tmp",
    width: 80,
    height: 12,
    active: false,
  }
  let state : SessionState = {
    name: "my-session",
    panes: [pane1, pane2],
    next_pane_id: 3,
  }
  assert_eq(state.name, "my-session")
  assert_eq(state.panes.length(), 2)
  assert_eq(state.next_pane_id, 3)
  assert_eq(state.panes[0].id, PaneId("%1"))
  assert_eq(state.panes[1].id, PaneId("%2"))
}

///|
/// SessionState: 空のペイン一覧で初期化
test "SessionState empty panes" {
  let state : SessionState = {
    name: "empty-session",
    panes: [],
    next_pane_id: 1,
  }
  assert_eq(state.panes.length(), 0)
  assert_eq(state.next_pane_id, 1)
}
