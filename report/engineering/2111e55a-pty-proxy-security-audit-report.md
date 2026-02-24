# PTY Proxy Security Audit Report

## Task: 2111e55a-pty-proxy-security-audit

## Summary

全6ソースファイル + 関連2ファイルを監査。9件の発見（H:1, M:3, L:3, Info:2）。

## Findings

| Severity | ID | Title |
|---|---|---|
| High | H-1 | /tmp symlink attack on socket directory (macOS標準では低リスク) |
| Medium | M-1 | PTY書き込みにエスケープインジェクション防御なし |
| Medium | M-2 | 状態ファイルディレクトリ権限の明示制御不足 |
| Medium | M-3 | CleanStaleSocket TOCTOU race |
| Low | L-1 | Wrapper一時ディレクトリ権限（確認済みで安全） |
| Low | L-2 | セッション名のソケットパス予測可能性 |
| Low | L-3 | umask復元がスレッドセーフでない |
| Info | I-1 | send-keysによるシェルコマンド注入は設計通り |
| Info | I-2 | 環境変数の信頼境界 |

## Positive Aspects

- umask(0077) でソケット作成: 適切
- ディレクトリ 0700: 適切
- 1MB リクエストサイズ制限: 適切
- セマフォ 16: DoS耐性あり
- 5秒接続タイムアウト: 適切
- Atomic state file writes: 適切

## Deliverable

`docs/drafts/security-audit.md` (186行)

## Commit

```
15ab84c docs: PTYプロキシ セキュリティ監査レポート
```
