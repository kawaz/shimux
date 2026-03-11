///|
/// tmux コマンドの解析結果
pub(all) enum TmuxCommand {
  Version
  HasSession(target~ : String)
  SwitchClient(target~ : String)
  NewSession(
    attach_if_exists~ : Bool,
    detached~ : Bool,
    session~ : String,
    cwd~ : String,
    command~ : Array[String]
  )
  ShowOptions(
    global~ : Bool,
    pane~ : Bool,
    value_only~ : Bool,
    target~ : String,
    name~ : String
  )
  SetOption(
    pane~ : Bool,
    window~ : Bool,
    unset~ : Bool,
    target~ : String,
    name~ : String,
    value~ : String
  )
  SplitWindow(
    horizontal~ : Bool,
    target~ : String,
    size~ : String,
    print_info~ : Bool,
    format~ : String
  )
  NewWindow(
    session~ : String,
    name~ : String,
    print_info~ : Bool,
    format~ : String
  )
  SendKeys(target~ : String, keys~ : Array[String])
  SelectPane(target~ : String, style~ : String, title~ : String)
  KillPane(target~ : String)
  ListPanes(format~ : String)
  DisplayMessage(print~ : Bool, format~ : String)
  Unknown(String)
} derive(Debug, Eq)

///|
/// assert_eq 互換のため Debug 経由で Show を提供（derive(Show) は v0.9 で非推奨）
pub impl Show for TmuxCommand with output(self, logger) {
  to_repr(self).output(logger)
}

///|
/// パースエラー
pub(all) suberror ParseError {
  MissingArgument(String)
  UnknownCommand(String)
  EmptyInput
}

///|
/// SplitWindow の direction を core の SplitDirection に変換する
pub fn split_direction(horizontal : Bool) -> @core.SplitDirection {
  if horizontal {
    Horizontal
  } else {
    Vertical
  }
}
