# Format String Compatibility Report

## Task: c404d405-format-string-compatibility

## Summary

tmuxフォーマット文字列互換性を拡張。13変数をサポートし、Claude Code が使用する主要変数を網羅。

## Changes

| File | Description |
|------|-------------|
| `internal/pane/manager.go` | Pane に PID, CurrentPath フィールド追加 + IndexOf() メソッド |
| `internal/pane/manager_test.go` | Save/Load round-trip, IndexOf テスト追加 |
| `internal/tmux/format.go` | FormatContext に PaneIndex, PaneTTY 追加 |
| `internal/tmux/format_test.go` | pane_tty, pane_index テスト追加 |
| `internal/tmux/commands.go` | buildFormatContext() 拡張 |
| `internal/tmux/commands_test.go` | 新変数5サブテスト追加 |
| `docs/drafts/format-variables.md` | 全13変数の仕様表 |

## Supported Format Variables (13)

| Variable | Source |
|----------|--------|
| pane_id | PaneInfo.ID |
| pane_pid | Pane.PID |
| pane_tty | Pane.TTY |
| pane_current_path | Pane.CurrentPath |
| pane_width | Pane.Width |
| pane_height | Pane.Height |
| pane_active | Pane.Active |
| pane_index | Manager.IndexOf() |
| session_name | GMUX_SESSION |
| session_id | "$0" (fixed) |
| window_id | "@0" (fixed) |
| window_index | "0" (fixed) |
| window_name | session_name |

## Commit

```
9fca198 feat: フォーマット変数拡張 — pane_pid/pane_tty/pane_index等対応
```
