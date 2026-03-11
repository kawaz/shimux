///|
/// ターミナル操作の抽象インターフェース
///
/// shim (tmux) と terminal (ghostty) の間の契約を定義する。
/// terminal バックエンドはこのインターフェースを実装し、
/// shim はこのインターフェースを通じて操作を発行する。
pub(open) trait Terminal {
  /// セッションが存在するか確認する
  has_session(Self, String) -> Bool raise TerminalError

  /// セッションを新規作成する
  new_session(Self, String, String, String) -> Unit raise TerminalError

  /// 新しいウィンドウを作成する
  new_window(Self, String, String) -> PaneId raise TerminalError

  /// ペインを分割する
  split_pane(Self, SplitDirection, String, Int) -> PaneId raise TerminalError

  /// ペインを分割してコマンドを実行する
  /// command: 実行コマンド配列, cwd: 作業ディレクトリ, env_vars: 環境変数 ["KEY=val", ...]
  split_pane_with_command(
    Self,
    SplitDirection,
    Array[String],
    String,
    Array[String],
  ) -> PaneId raise TerminalError

  /// 指定ペインにキーを送信する
  send_keys(Self, PaneId, String) -> Unit raise TerminalError

  /// 指定ペインにテキストを送信する（input text 用）
  send_text(Self, PaneId, String) -> Unit raise TerminalError

  /// 指定ペインを選択（フォーカス）する
  select_pane(Self, PaneId) -> Unit raise TerminalError

  /// 指定ペインを削除する
  kill_pane(Self, PaneId) -> Unit raise TerminalError

  /// ペイン一覧を取得する
  list_panes(Self, String) -> Array[PaneInfo] raise TerminalError

  /// フォーマット変数を展開してメッセージを返す
  display_message(Self, String) -> String raise TerminalError

  /// オプション値を取得する
  show_option(Self, String) -> String raise TerminalError

  /// ペインをリサイズする
  resize_pane(Self, PaneId, String, Int) -> Unit raise TerminalError
}
