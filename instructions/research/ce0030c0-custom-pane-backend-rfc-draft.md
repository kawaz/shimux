# CustomPaneBackend RFC草案の作成

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査＋設計タスク。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
Claude Code Issue #26572 で提案されているCustomPaneBackendプロトコルのRFC草案を作成する。gmuxのPoC実装の知見を踏まえ、実現可能で実用的なプロトコル仕様を起草する。

## 作業内容

1. **既存提案の調査**
   - Claude Code Issue #26572の最新状況をWebSearchで確認
   - KILDのCustomPaneBackend実装の調査
   - 関連するフィードバックやコメントの収集

2. **gmux PoC実装からのフィードバック**
   - wip-redesignのPoC実装を参照し、tmux shimで困った点をリスト化
   - tmux互換を保つことの難しさ（フォーマット文字列、引数パース等）
   - PTYプロキシで解決できたこととできなかったこと
   - 「こうあれば良かった」というAPI設計のインサイト

3. **RFC草案の作成**
   - JSON-RPC 2.0ベースのプロトコル仕様
   - 7操作の詳細定義:
     - `spawn_agent`: 新ペイン作成
     - `write`: データ書き込み（send-keys相当）
     - `capture`: 画面内容取得
     - `kill`: ペイン終了
     - `list`: ペイン一覧
     - `get_self_id`: 自身のペインID取得
     - `context_exited`: コンテキスト終了通知
   - エラーコードの定義
   - バージョニング戦略
   - 環境変数 `CLAUDE_PANE_BACKEND` の仕様

4. **gmux二重対応の設計**
   - tmux shim モード（現行）
   - CustomPaneBackend モード（将来）
   - 切り替え/共存の仕組み

## 期待する成果物
- RFC草案（`docs/drafts/custom-pane-backend-rfc.md`）
- gmuxからのフィードバックメモ
- 実装ロードマップ案
