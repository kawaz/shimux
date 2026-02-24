# ペインライフサイクル状態マシンの設計

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
gmuxにおけるペインの状態遷移を正式な状態マシンとして定義する。split-window→active→send-keys→kill-paneの一連のライフサイクルを厳密にモデル化し、不正な状態遷移を検出できるようにする。

## 設計項目

1. **状態の定義**
   - `creating`: split-window発行中（keysim実行中）
   - `initializing`: gmux-agent起動待ち（ソケット接続待ち）
   - `ready`: send-keys受付可能
   - `busy`: 処理中（長時間コマンド実行中等）
   - `closing`: kill-pane発行中
   - `dead`: 終了済み（ソケット切断確認済み）
   - `orphaned`: gmux-agentプロセスが予期しない終了

2. **状態遷移の定義**
   - 各状態から許可される遷移を定義
   - 不正な遷移の検出と対応（ログ出力 or エラー返却）
   - タイムアウトによる自動遷移（creating→orphaned等）

3. **状態ファイルとの整合性**
   - `panes-<session>.json` にstate fieldを追加
   - 起動時の状態復元ロジック
   - ファイルと実プロセスの不整合検出

4. **テスト設計**
   - 各状態遷移のテストケース
   - 不正遷移の検出テスト
   - タイムアウト処理のテスト
   - property-based testingの検討

## 期待する成果物
- 状態遷移図（Mermaid記法）
- Go interfaceとenum定義
- テストケース一覧
- 設計書への追記案
