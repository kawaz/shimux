# Escape Sequence Handling Report

## Task: d12fa59c-escape-sequence-handling

## Summary

tmux互換キーマッピングの調査・実装。specialKeysマップに51エントリ追加、修飾キー(C-/M-/S-)対応、xterm修飾パラメータ対応を実装。

## Changes

| File | Lines | Description |
|------|-------|-------------|
| `internal/tmux/keys.go` | +336 | 51エントリ追加、修飾キー解析、xterm修飾パラメータ |
| `internal/tmux/keys_test.go` | +380 | 約120新規テスト |
| `docs/drafts/key-mapping.md` | +166 | キーマッピング表 + Claude Code互換性 |

## Key Additions

- **エイリアス**: Insert, Delete, PageDown, PgDn, PageUp, PgUp
- **C0制御文字**: NUL〜US (29個)
- **テンキー**: KP0-KP9, KP/, KP*, KP-, KP+, KP., KPEnter (16個)
- **修飾キー**: C-A〜C-Z (大文字), C-@, M-x, S-x, C-M-x, C-M-S-x
- **修飾付き特殊キー**: C-Up, S-F1, M-Left等 (xterm修飾パラメータ形式)

## Commit

```
b8ae1ec feat: キーマッピング大幅拡張 — tmux互換性向上
```
