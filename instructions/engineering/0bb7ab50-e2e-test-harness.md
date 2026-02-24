# E2Eテストハーネスの設計

## 指示
承認不要の完全オートモードで以下を行うこと。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
gmuxの全体的なE2E（End-to-End）テストを実行するためのテストハーネスを設計する。実際のGhostty環境がなくても、gmuxの主要なフローをモック環境で検証できる仕組みを構築する。

## 設計項目

1. **テストハーネスのアーキテクチャ**
   - keysimモジュールのモック（System Events呼び出しを記録するだけ）
   - gmux-agentのモック（実PTYの代わりにpipe使用）
   - 状態ファイルの一時ディレクトリ隔離
   - ソケット通信のloopback構成

2. **テストシナリオ**
   - **基本フロー**: new-session → split-window → send-keys → capture-pane → kill-pane → kill-session
   - **Claude Codeシミュレーション**: Agent Teamsの典型的なtmuxコマンド列を再現
   - **エラーフロー**: 存在しないペインへの操作、同時split、orphaned pane

3. **テストインターフェース設計**
   ```go
   type TestHarness struct {
       KeysimRecorder  *KeysimRecorder  // キーシミュレーション記録
       AgentPool       *MockAgentPool   // モックgmux-agent群
       StateDir        string           // 一時状態ディレクトリ
   }
   ```

4. **CIでの実行**
   - macOS runner不要（モックで完結）
   - Linux runnerでも実行可能
   - `go test -tags=e2e ./e2e/...` でのゲート
   - テスト実行時間の目標: 30秒以内

## テスト優先度
1. split-window + list-panes のペインID取得フロー（Critical）
2. send-keys + capture-pane のデータI/Oフロー（Critical）
3. kill-pane + 状態更新のクリーンアップフロー（High）
4. セッション管理フロー（High）
5. エラー・リカバリーフロー（Medium）

## 期待する成果物
- テストハーネスの設計書（`docs/drafts/e2e-test-harness.md`）
- Go interfaceの定義
- テストシナリオのテーブル
