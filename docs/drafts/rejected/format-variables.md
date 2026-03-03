# gmux Format Variables

tmux互換フォーマット変数の一覧。`#{variable_name}` 構文で展開される。

## サポート済み変数

| Variable | Type | Description | Source |
|---|---|---|---|
| `session_name` | string | セッション名 | `CommandContext.Session` |
| `session_id` | string | セッションID (`$0` 固定) | 固定値 |
| `window_id` | string | ウィンドウID (`@0` 固定) | 固定値 |
| `window_index` | string | ウィンドウインデックス (`0` 固定) | 固定値 |
| `window_name` | string | ウィンドウ名 (セッション名と同一) | `CommandContext.Session` |
| `pane_id` | string | ペインID (`%N` 形式) | `Pane.ID` |
| `pane_index` | string | ペインの登録順インデックス | `Manager.IndexOf()` |
| `pane_pid` | int | ペイン内プロセスのPID | `Pane.PID` |
| `pane_tty` | string | PTYデバイスパス (例: `/dev/ttys005`) | `Pane.TTY` |
| `pane_current_path` | string | カレントディレクトリ | `Pane.CurrentPath` |
| `pane_width` | int | ペイン幅 (カラム数) | `Pane.Width` |
| `pane_height` | int | ペイン高さ (行数) | `Pane.Height` |
| `pane_active` | bool | アクティブフラグ (`1`/`0`) | `Pane.Active` |

## 固定値について

gmuxはGhosttyの単一ウィンドウ内で動作するため、以下の変数は固定値を返す:
- `session_id`: 常に `$0`
- `window_id`: 常に `@0`
- `window_index`: 常に `0`

## 未知の変数

未知の変数名は `#{unknown_name}` のまま展開されずに返される（tmux互換動作）。

## 使用例

```bash
# ペイン一覧をカスタムフォーマットで表示
gmux list-panes -F '#{pane_index}: #{pane_id} [#{pane_width}x#{pane_height}] #{pane_tty}'

# アクティブペインのPIDを取得
gmux display-message -p '#{pane_pid}'

# split-window で新ペインIDを取得
gmux split-window -P -F '#{pane_id}'
```
