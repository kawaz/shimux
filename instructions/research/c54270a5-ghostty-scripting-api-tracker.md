# Ghostty Scripting API進捗トラッカーと影響分析

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査タスク。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
Ghostty Scripting API (ghostty-org/ghostty#2353) の最新の開発状況を調査し、gmuxへの影響を分析する。このAPIが実装されればkeysimによるペイン操作が不要になり、gmuxのアーキテクチャに大きな影響を与える可能性がある。

## 調査項目

1. **Issue #2353 の最新状況**
   - WebSearchでghostty scripting APIの最新情報を調査
   - PRやブランチでの進捗確認
   - 開発者のコメントからのタイムライン推測
   - 関連するRFC/提案ドキュメント

2. **提案されているAPI仕様**
   - 利用可能な操作（split, close, focus, send-text等）
   - 通信方式（Unix socket? stdin/stdout? DBus?）
   - 認証/認可モデル
   - macOS対応の予定

3. **gmuxへの影響分析**
   - Scripting APIが利用可能になった場合のアーキテクチャ変更案
   - keysimモジュールの置き換え範囲
   - PTYプロキシとの共存/置換の判断基準
   - 移行戦略（keysim→Scripting API）

4. **暫定対応方針**
   - APIが未実装の間のkeysim維持
   - 将来の切り替えを容易にする抽象化レイヤー
   - TerminalBackend interface の設計（代替ターミナル調査結果との統合）

## 期待する成果物
- Scripting API状況レポート（`docs/drafts/ghostty-scripting-api-status.md`）
- gmux影響分析と移行戦略案
- アーキテクチャ抽象化レイヤーの設計スケッチ
