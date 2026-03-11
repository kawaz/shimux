///|
/// Ghostty AppleScript バックエンドの状態
pub(all) struct GhosttyBackend {
  /// Ghostty アプリケーション名（AppleScript の tell 対象）
  app_name : String
} derive(Debug)

///|
/// GhosttyBackend のコンストラクタ
pub fn GhosttyBackend::new() -> GhosttyBackend {
  { app_name: "Ghostty" }
}

///|
/// GhosttyBackend が Terminal trait を実装していることを確認するヘルパー。
/// 外部パッケージが Terminal trait 経由で GhosttyBackend を使う際の型チェック用。
pub fn GhosttyBackend::as_terminal(self : GhosttyBackend) -> &@core.Terminal {
  self
}
