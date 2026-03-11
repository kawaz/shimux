///|
/// ペインの一意識別子
pub(all) struct PaneId(String) derive(Eq, Debug)

///|
/// assert_eq 互換のため Debug 経由で Show を提供（derive(Show) は v0.9 で非推奨）
pub impl Show for PaneId with output(self, logger) {
  to_repr(self).output(logger)
}

///|
/// ペインの情報
pub(all) struct PaneInfo {
  id : PaneId
  name : String
  cwd : String
  width : Int
  height : Int
  active : Bool
}

///|
/// ペイン分割の方向
pub(all) enum SplitDirection {
  Horizontal
  Vertical
}

///|
/// ターミナル操作のエラー
pub(all) suberror TerminalError {
  TerminalError(String)
}

///|
/// tmux フォーマット変数
pub(all) enum FormatVar {
  SessionName
  WindowIndex
  PaneId
  PanePid
  PaneCurrentPath
  PaneWidth
  PaneHeight
  PaneActive
}

///|
/// セッション状態
pub(all) struct SessionState {
  name : String
  panes : Array[PaneInfo]
  next_pane_id : Int
}
