# PTYプロキシ セキュリティ監査

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
gmux-agentのPTYプロキシ設計についてセキュリティ観点から徹底的に監査する。

## 作業内容

1. **Unixソケット権限の調査**
   - `~/.local/state/gmux/` 配下のソケットファイルのパーミッション設計を確認
   - 他ユーザーからのソケット接続可否を分析
   - tmpdir利用時のsymlink attack耐性を検討

2. **PTY書き込みの安全性**
   - `master.Write(data)` で注入可能なエスケープシーケンスの列挙
   - ターミナルエスケープインジェクション攻撃の可能性を調査
   - 入力サニタイズの必要性と方針を検討

3. **セッション管理**
   - 状態ファイル `panes-<session>.json` のrace condition耐性
   - staleソケットの検出と安全なクリーンアップ方法
   - gmux-agentプロセスの権限昇格可能性

4. **レポート作成**
   - 発見した問題点をseverity(Critical/High/Medium/Low)で分類
   - 各問題に対する修正提案を設計書のDR形式で記述
   - `docs/drafts/security-audit.md` として出力

## 期待する成果物
- セキュリティ監査レポート（`docs/drafts/security-audit.md`）
- 必要に応じて設計書への修正提案PR

> **配置ルール**: この指示による新規ドキュメントは `docs/drafts/` 配下に配置すること。既存ファイルの編集は直接行ってよい。
