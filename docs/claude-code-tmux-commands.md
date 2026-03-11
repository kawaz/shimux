# Claude Code tmux コマンド調査結果

調査日: 2026-03-12
調査対象: Claude Code v2.1.74
調査方法: npm パッケージのソースコード解析 + ロギング shim による実行時キャプチャ

## tmux コマンド一覧（21 種類）

### セッション管理

| コマンド | 呼び出し形式 | 目的 |
|---|---|---|
| `-V` | `tmux -V` | tmux 存在確認 |
| `has-session` | `tmux has-session -t <session>` | セッション存在確認 |
| `new-session` | `tmux new-session -d -s <session> -c <cwd>` | セッション作成 (detached) |
| | `tmux new-session -d -s <session> -c <cwd> -- <cmd>` | コマンド付き作成 |
| | `tmux [-CC] new-session -A -s <session> -c <cwd> -- <cmd>` | attach-if-exists |
| | `tmux -L <socket> new-session -d -s <session> -n <window> -P -F "#{pane_id}"` | swarm 用 |
| `kill-session` | `tmux kill-session -t <session>` | セッション破棄 |
| `switch-client` | `tmux switch-client -t <session>` | セッション切替 |
| `attach-session` | `tmux [-CC] attach-session -t <session>` | セッション接続 |

### ペイン管理

| コマンド | 呼び出し形式 | 目的 |
|---|---|---|
| `split-window` | `tmux split-window -t <pane> -h -l 70% -P -F "#{pane_id}"` | 最初の teammate |
| | `tmux split-window -t <pane> [-h\|-v] -P -F "#{pane_id}"` | 追加 teammate |
| `new-window` | `tmux new-window -t <session> -n <name> -P -F "#{pane_id}"` | 新ウィンドウ |
| `kill-pane` | `tmux kill-pane -t <pane>` | ペイン破棄 |
| `select-pane` | `tmux select-pane -t <pane>` | フォーカス |
| | `tmux select-pane -t <pane> -P "bg=default,fg=<color>"` | スタイル設定 |
| | `tmux select-pane -t <pane> -T <title>` | タイトル設定 |
| `send-keys` | `tmux send-keys -t <pane> <cmd_string> Enter` | コマンド送信 |
| `list-panes` | `tmux list-panes -t <window> -F "#{pane_id}"` | ペイン一覧 |
| `resize-pane` | `tmux resize-pane -t <pane> -x 30%` | リサイズ |
| `break-pane` | `tmux break-pane -d -s <pane> -t <hidden_session>:` | ペイン非表示 |
| `join-pane` | `tmux join-pane -h -s <pane> -t <window>` | ペイン復帰 |

### レイアウト・スタイル

| コマンド | 呼び出し形式 | 目的 |
|---|---|---|
| `set-option` | `tmux set-option -p -t <pane> pane-border-style "fg=<color>"` | ボーダー色 |
| | `tmux set-option -p -t <pane> pane-active-border-style "fg=<color>"` | アクティブボーダー色 |
| | `tmux set-option -p -t <pane> pane-border-format "..."` | ボーダーフォーマット |
| | `tmux set-option -w -t <window> pane-border-status top` | ボーダーステータス |
| `select-layout` | `tmux select-layout -t <window> main-vertical` | レイアウト設定 |
| | `tmux select-layout -t <window> tiled` | タイルレイアウト |
| `show-options` | `tmux show-options -g prefix` | prefix キー取得 |
| | `tmux show-options -p -v window-active-style` | ペインスタイル取得 |

### その他

| コマンド | 呼び出し形式 | 目的 |
|---|---|---|
| `display-message` | `tmux display-message -p "#{pane_id}"` | 現在ペイン ID 取得 |
| | `tmux display-message -t <pane> -p "#{session_name}:#{window_index}"` | ウィンドウターゲット取得 |
| `list-windows` | `tmux list-windows -t <session> -F "#{window_name}"` | ウィンドウ一覧 |
| `load-buffer` | `echo <content> \| tmux load-buffer -w -` | クリップボード |

## 呼び出しフロー

### 1. `--tmux` セッション起動

```
tmux -V
tmux show-options -g prefix
tmux has-session -t <session>
tmux new-session -A -s <session> -c <cwd> -- <claude>
```

### 2. Worktree 起動後のスタイル設定

```
tmux show-options -p -v window-active-style
tmux set -p window-active-style bg=#404040
# 終了時:
tmux set -p -u window-active-style
```

### 3. Teammate ペイン作成（tmux 内部、swarm view）

```
tmux display-message -p "#{pane_id}"
tmux display-message -t <pane> -p "#{session_name}:#{window_index}"
tmux list-panes -t <window> -F "#{pane_id}"
tmux split-window -t <leader> -h -l 70% -P -F "#{pane_id}"
tmux select-pane -t <pane> -P "bg=default,fg=<color>"
tmux set-option -p -t <pane> pane-border-style "fg=<color>"
tmux set-option -p -t <pane> pane-active-border-style "fg=<color>"
tmux select-pane -t <pane> -T <name>
tmux set-option -p -t <pane> pane-border-format "..."
tmux set-option -w -t <window> pane-border-status top
tmux select-layout -t <window> main-vertical
tmux resize-pane -t <first_pane> -x 30%
tmux send-keys -t <pane> "cd ... && env ... claude ..." Enter
```

### 4. ペインの hide/show

```
# Hide:
tmux new-session -d -s claude-hidden
tmux break-pane -d -s <pane> -t claude-hidden:

# Show:
tmux join-pane -h -s <pane> -t <window>
tmux select-layout -t <window> main-vertical
tmux resize-pane -t <first_pane> -x 30%
```

### 5. 終了時

```
tmux kill-pane -t <pane>
tmux kill-session -t <session>
```

## 環境変数

| 変数 | 用途 |
|---|---|
| `TMUX` | tmux 内で実行中か判定 |
| `TMUX_PANE` | 現在のペイン ID |
| `CLAUDE_CODE_TMUX_SESSION` | worktree 用 tmux セッション名 |
| `CLAUDE_CODE_TMUX_PREFIX` | tmux の prefix キー |
| `CLAUDE_CODE_TMUX_PREFIX_CONFLICTS` | prefix コンフリクトフラグ |

## フォーマット変数

| 変数 | 使用箇所 |
|---|---|
| `#{pane_id}` | split-window, new-window, new-session, list-panes, display-message |
| `#{session_name}` | display-message |
| `#{window_index}` | display-message |
| `#{window_name}` | list-windows |
| `#{pane_title}` | pane-border-format |

## `-L <socket>` パターン

外部 swarm モード（tmux 外からの起動時）では専用ソケットを使用:
- ソケット名: `claude-swarm-${process.pid}`
- セッション名: `claude-swarm`
- ウィンドウ名: `swarm-view`
- 非表示セッション名: `claude-hidden`

## shimux の現状との差分

### 未実装コマンド（高優先度）

- `kill-session -t <session>`
- `set-option -p/-w -t <target> <option> <value>` (`set` エイリアス含む)
- `select-layout -t <target> <layout>`
- `break-pane -d -s <source> -t <target>`
- `join-pane -h -s <source> -t <target>`

### 未実装コマンド（中優先度）

- `list-windows -t <session> -F <format>`
- `attach-session -t <session>`
- `resize-pane -t <pane> -x <size>` (trait にはあるがパーサにない)

### 未実装コマンド（低優先度）

- `load-buffer -w -` (クリップボード)
- `-CC` フラグ (iTerm2 統合)

### パーサ拡張が必要

- `new-session`: `-d`, `-n`, `-P`, `-F` フラグ追加
- `list-panes`: `-t <target>` オプション追加
- `display-message`: `-t <target>` オプション追加
- `show-options`: `-p`, `-v` のペインレベル対応
- `split-window`: `-v` の明示的ハンドリング
