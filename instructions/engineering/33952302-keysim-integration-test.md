# keysim統合テストの追加

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
System Events経由のキーシミュレーション（keysim）のユニットテストと統合テストを設計・実装する。

## 作業内容

1. **keysimモジュールの調査**
   - 現在の `keysim` 実装を読み、テスト可能な境界を特定
   - macOS System Events APIの呼び出し箇所を特定
   - モック可能なインターフェースを設計

2. **ユニットテスト作成**
   - キーコードマッピングの正確性テスト（Cmd+D, Cmd+Shift+Enter等）
   - osascript コマンド組み立てのテスト
   - エラーハンドリング（System Events権限なし等）のテスト

3. **統合テストの設計**
   - CI環境（headless）での実行可否を調査
   - テスト用のモックターミナルバックエンド設計
   - `go test -tags=integration` でのゲート付きテスト

4. **テストヘルパー**
   - `testutil/keysim_mock.go` の作成
   - テストフィクスチャの整備
   - テストカバレッジ目標: 80%以上

## テスト対象コマンド
- `split-window -v` → Cmd+D（下分割）
- `split-window -h` → Cmd+Shift+D（右分割）
- `kill-pane` → Cmd+W（ペイン閉じ）
- `select-pane -t next` → Cmd+]（次のペイン）
- `select-pane -t previous` → Cmd+[（前のペイン）

## 期待する成果物
- テストコード（`*_test.go`）
- テストヘルパー（`testutil/`）
- テストカバレッジレポート
- テスト実行手順のドキュメント更新
