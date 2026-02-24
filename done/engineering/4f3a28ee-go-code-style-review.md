# Go コードスタイル・品質レビュー

## 指示
承認不要の完全オートモードで以下を行うこと。/itumono-full-review を使用すること。

## 概要
gmuxプロジェクトの全Goコードに対して、Goの慣用的な書き方（idiomatic Go）に準拠しているかをfull-reviewする。

## レビュー観点

1. **命名規則**
   - パッケージ名、関数名、変数名がGo Effective Go準拠か
   - エクスポートされた名前のドキュメントコメントの有無
   - 略語の扱い（URL, HTTP, ID等の大文字ルール）

2. **エラーハンドリング**
   - `error` の適切なラッピング（`fmt.Errorf` with `%w`）
   - sentinel error vs error type の使い分け
   - `errors.Is` / `errors.As` の利用

3. **並行処理**
   - goroutineのリーク防止（context.Cancel, done channel）
   - sync.Mutex vs channel の選択
   - `-race` フラグでの検出可能性

4. **テスタビリティ**
   - インターフェースの活用度
   - テーブル駆動テストの一貫性
   - テストヘルパーの `t.Helper()` 呼び出し

5. **パフォーマンス**
   - 不要なアロケーション
   - string/byte変換の効率
   - バッファサイズの妥当性

## 期待する成果物
- コードレビューレポート
- リファクタリング提案のPR候補
- `golangci-lint` 設定ファイル（`.golangci.yml`）の提案
