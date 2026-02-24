# 実機Ghostty環境での手動テスト計画

## 指示
承認不要の完全オートモードで以下を行うこと。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
PoC実装がmainマージ段階に近づいている。モックテストは充実しているが、実際のGhostty環境での手動テスト手順書を作成する。ユーザーが実機で確認する際のガイドとなる。

## テスト手順書の作成

### Phase 1: 基本動作確認
1. **gmux-agent起動確認**
   - ghostty configに `command = gmux-agent` を設定
   - Ghostty再起動後、シェルが正常に起動するか
   - `gmux-agent` プロセスの存在確認（`ps aux | grep gmux-agent`）
   - Unixソケットの生成確認

2. **ペイン操作**
   - `gmux split-window -v` で下に分割されるか
   - `gmux split-window -h` で右に分割されるか
   - `gmux kill-pane -t %1` でペインが閉じるか
   - `gmux select-pane -t next` / `-t previous` でフォーカスが移動するか
   - keysimの遅延は許容範囲内か（体感500ms以下）

3. **データI/O**
   - `gmux send-keys -t %0 "echo hello" Enter` でコマンドが実行されるか
   - `gmux capture-pane -t %0 -p` で画面内容が取得できるか
   - 特殊キー（Ctrl+C, Tab等）が正しく送信されるか

### Phase 2: Claude Code統合テスト
1. **tmux偽装モード**
   - `gmux -- claude --worktree --tmux` で起動するか
   - Claude Codeがtmux環境を認識するか
   - Agent Teamsの split/send-keys/kill が動作するか

2. **セッション管理**
   - セッション名が正しく生成されるか
   - `gmux has-session` / `gmux list-sessions` が正しく応答するか

### Phase 3: 負荷・安定性
1. 5分間連続使用での安定性
2. 10ペイン同時運用での挙動
3. Ghosttyウィンドウリサイズ時の挙動

## 期待する成果物
- 手動テスト手順書（`docs/drafts/manual-test-guide.md`）
- テスト結果記入テンプレート
- 既知の制限事項リスト
