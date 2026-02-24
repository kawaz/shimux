# tmux互換性マトリクスの作成と検証

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
Claude Codeが使用する全tmuxコマンドとフラグの組み合わせを網羅的に調査し、gmuxでの互換性マトリクスを作成する。

## 作業内容

1. **Claude Codeソースコード調査**
   - Claude Codeがtmuxを呼び出す全箇所を特定（WebSearch活用）
   - 各コマンドの引数パターンを列挙
   - 呼び出し順序とエラーハンドリングを分析

2. **tmux実際の挙動記録**
   - 各コマンドのstdout/stderr出力の正確な形式を記録
   - exit codeの値を記録
   - 異常系（存在しないペイン指定等）の挙動を記録

3. **互換性マトリクス作成**
   - コマンド × フラグの全組み合わせ表
   - gmux実装状態: Implemented / Planned / Not Needed / N/A
   - 互換性レベル: Full / Partial / Stub / None
   - 優先度: Critical / High / Medium / Low

4. **テストケース自動生成**
   - マトリクスから `go test` のテーブル駆動テストを自動生成するスクリプト
   - tmux実機との差異を検出するテスト

## 期待する成果物
- 互換性マトリクス（`docs/drafts/compatibility-matrix.md`）
- テストケース生成スクリプト
- 未実装コマンドのIssue候補リスト

> **配置ルール**: この指示による新規ドキュメントは `docs/drafts/` 配下に配置すること。既存ファイルの編集は直接行ってよい。
