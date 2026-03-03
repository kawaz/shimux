# IPC 方式の調査（2026-02-25）

## 結論

**Unix domain socket が事実上の標準**。shimux でも採用して問題ない。

## 既存ターミナルマルチプレクサの IPC

| ツール | IPC | プロトコル | 備考 |
|---|---|---|---|
| tmux | Unix domain socket | OpenBSD imsg (バイナリ) | MAX_IMSGSIZE=16384, SCM_RIGHTS でfd受渡し |
| GNU screen | Unix domain socket (FIFO も可) | 独自バイナリ | パーミッションビットでセッション状態表示 |
| zellij | Unix domain socket | Protocol Buffers | length-prefixed framing |
| abduco/dtach | Unix domain socket | 独自軽量バイナリ | detach 機能特化 |

## Unix domain socket が選ばれる理由

- 双方向通信が1本で完結
- `SCM_RIGHTS` で PTY fd をプロセス間転送可能
- ファイルパーミッションで自然にアクセス制御（追加認証不要）
- カーネル内バッファコピーのみで低オーバーヘッド

## shimux での設計判断

- **パス長制限**: macOS `sun_path` 104バイト → セッション名をハッシュ化すれば問題なし
- **プロトコル**: TTY データは PTY read 単位（4KB〜16KB）で細切れに届く。length-prefixed バイナリ (`[4B length][payload]`) で十分。tmux も MAX_IMSGSIZE=16384 で回っている
- **tmux との共存**: tmux の内部プロトコル(imsg)を直接話す必要なし。`posix_spawn` で tmux コマンドを呼ぶ方式（PoC 確認済み）で OK

## 不採用の代替案

| 代替 | 却下理由 |
|---|---|
| Named pipe (FIFO) | 双方向に2本必要。柔軟性で劣る |
| TCP socket | ローカル用途には過剰。認証・暗号化の複雑化 |
| D-Bus | macOS で不向き |
| Shared memory | 実装の複雑さに見合わない |
