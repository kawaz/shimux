# KILD (kild-tmux-shim) との比較研究

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査タスク。

## 概要
KILDのkild-tmux-shimはgmuxと同様のアプローチ（tmux shimによるターミナル互換化）を採用している。両プロジェクトを比較研究し、gmuxに活かせる知見を抽出する。

## 調査内容

1. **KILDアーキテクチャ調査**
   - kild-tmux-shimのソースコード調査（GitHub検索）
   - 実装言語とアーキテクチャ
   - 対応tmuxコマンドの一覧
   - PTY管理の実装方式

2. **設計判断の比較**
   - ペインID管理方式
   - 状態永続化の方法
   - エラーハンドリング戦略
   - テスト戦略

3. **gmuxとの差異分析**
   | 観点 | gmux | KILD |
   |------|------|------|
   | 対象ターミナル | Ghostty | ? |
   | send-keys方式 | PTYプロキシ | ? |
   | ペイン操作 | System Events | ? |
   | 言語 | Go | ? |
   | CustomPaneBackend | 計画中 | 先行実装 |

4. **学べること**
   - gmuxに取り入れるべき設計パターン
   - KILDが解決済みでgmuxが未解決の問題
   - 両プロジェクトの協力可能性

## 期待する成果物
- 比較研究レポート（`docs/drafts/kild-comparison.md`）
- gmuxへのフィードバックリスト
- CustomPaneBackend互換性の評価

> **配置ルール**: この指示による新規ドキュメントは `docs/drafts/` 配下に配置すること。既存ファイルの編集は直接行ってよい。
