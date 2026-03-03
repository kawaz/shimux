# shimux MoonBit 実装設計書

## shimux とは

Ghostty 上で Claude Code Agent Teams（tmux モード）を動作させるための **tmux 互換 CLI ラッパー**。tmux のフル互換ではなく、Claude Code が必要とするコマンドサブセットに特化する。

## セキュリティモデル

shimux は **同一 UID の同一ホスト内で動作するユーザーツール** である。信頼境界は Unix のユーザー分離に依存する。

- ソケットディレクトリのパーミッション 0700 で同一 UID に制限
- ソケットパスは `$XDG_RUNTIME_DIR` または `$TMPDIR/shimux-$UID/` 配下に配置
- `/tmp` 直下は使わない（TOCTOU リスク回避）

### DR: SO_PEERCRED による追加検証を行わない理由

ソケットディレクトリが 0700 であれば、同一 UID 以外のプロセスは connect できない。SO_PEERCRED による追加検証は冗長であり、macOS では対応する API（LOCAL_PEERCRED）の挙動も異なるため、クロスプラットフォームの複雑さに見合わない。

### DR: LD_PRELOAD 等のフィルタリングを行わない理由

shimux 自体が信頼できる環境で起動される前提。LD_PRELOAD を注入できるプロセスはすでに同一 UID で任意コード実行可能であり、フィルタリングに実効性がない。

## 技術スタック

```
MoonBit (native backend) — ロジック層
    └── extern "C" → Rust staticlib (C ABI) — システム層
                          └── libc クレート → POSIX API
```

### DR: なぜ MoonBit + Rust か

| 選択肢 | 評価 |
|---|---|
| Go（現 PoC） | 動作するが、shimux 本体の実装言語としては MoonBit を検討中 |
| MoonBit + Node.js FFI | TTY/PTY は Node.js の node-pty 依存。配布が面倒 |
| MoonBit + C FFI | 動くが C は避けたい |
| **MoonBit + Rust FFI** | Rust で安全に POSIX API をラップ、C ABI で公開。**採用** |

### DR: なぜ native backend か

shimux は PTY・Unix socket・シグナル等の POSIX API を直接使う。JS/Wasm ターゲットでは不可能。native backend の `extern "C"` でのみ実現可能。

## アーキテクチャ

### 全体構成（Go PoC と同一構造を MoonBit で再実装）

```
shimux (argv[0]="tmux" via symlink)
  ├── CLI 層: tmux 互換の引数解析
  ├── コマンド変換層: tmux コマンド → 内部アクション
  ├── バックエンド層: Ghostty への実操作（keysim / osascript）
  └── エージェント層: shimux-agent（PTY プロキシ）

shimux-agent（各ペインで起動）:
  Ghostty PTY (master/slave)
    └── shimux-agent
        ├── 内部 PTY (master/slave) → $SHELL
        ├── I/O プロキシ (Ghostty PTY ⟷ 内部 PTY)
        ├── Unix ソケット (send-keys 受信)
        ├── Observer (出力監視、capture-pane 用)
        └── SIGWINCH 転送
```

### データフロー

```
[Ghostty] keyboard
    ↓ stdin
[shimux-agent]
    ↓ I/O プロキシ → observer.on_input()
    ↓ master_fd write
[内部 PTY slave] → $SHELL
    ↓ stdout/stderr
[内部 PTY master]
    ↓ master_fd read → observer.on_output()
[shimux-agent]
    ↓ stdout write
[Ghostty] 表示

--- 別経路 (send-keys) ---

[shimux send-keys -t %0 "echo hi" Enter]
    ↓ Unix socket connect
[shimux-agent socket server]
    ↓ observer.on_input()
    ↓ master_fd write
[$SHELL] 受信

--- SIGWINCH 転送 ---

[Ghostty] ウィンドウリサイズ
    ↓ SIGWINCH
[shimux-agent] sig_setup_winch ハンドラ
    ↓ TIOCGWINSZ(stdin) → TIOCSWINSZ(master_fd)
[内部 PTY slave] → $SHELL (SIGWINCH 自動送信)
```

DR: send-keys 経由の入力も `observer.on_input()` を通す理由 — capture-pane の画面バッファ整合性を保つため。send-keys で送られた入力のエコーバックは on_output で捕捉されるが、入力自体も on_input で追跡できることで、将来の入力ログ機能にも対応できる。

### Claude Code Agent Teams が使う tmux コマンド

Go PoC の実装から確認済み:

| コマンド | 用途 |
|---|---|
| `has-session -t <name>` | セッション存在確認 |
| `new-session -d -s <name> -x W -y H` | セッション作成 |
| `show-options -g prefix` | prefix キー確認 |
| `split-window -t <pane> -h/-v -P -F "#{pane_id}"` | ペイン分割 |
| `send-keys -t <pane> "<text>" Enter` | キー送信 |
| `select-pane -t <pane>` | ペインフォーカス |
| `list-panes -t <session> -F "#{pane_id} ..."` | ペイン一覧 |
| `kill-pane -t <pane>` | ペイン終了 |

フォーマット変数: `#{pane_id}`, `#{pane_active}`, `#{session_name}`

## FFI 設計

### 設計原則

1. **カテゴリプレフィックス**: `shimux_{category}_{verb}` で名前空間を整理
2. **1 関数 1 責務**: 神関数を作らない
3. **対称ペア**: open/close, save/restore
4. **不透明ハンドル = Int**: MoonBit FFI で構造体やポインタは渡せない。ハンドル（Rust 側の内部テーブルのインデックス）を Int で返す
5. **複数値返却 = Int64 パッキング**: MoonBit FFI は戻り値 1 つ。2 値は Int64 にパックする
6. **文字列引数 = Bytes (UTF-8)**: MoonBit 側で `@utf8.encode()` してから渡す
7. **テスタブル**: 各関数を単独でテスト可能な粒度に分離

### 戻り値スキーマ

全 FFI 関数の戻り値は以下の 5 パターンに分類される。各関数の説明では略記する。

| パターン | 意味 | 適用例 |
|---|---|---|
| **0/−1** | 成功=0、失敗=−1 | `pty_resize`, `tty_set_raw`, `io_close` |
| **Boolean** | true=1、false=0 | `tty_is_tty` |
| **Tristate** | 1=真、0=偽、−1=エラー | `daemon_pid_check`(1=生存/0=不在/−1=エラー) |
| **Handle** | 0以上=ハンドル、−1=失敗 | `pty_open`, `tty_save`, `sock_listen` |
| **Size** | 0以上=バイト数、0=EOF、−1=エラー | `io_read`, `io_write` |

### DR: 不透明ハンドルを Int にする理由

MoonBit の `extern "C"` は Int, Int64, Double, Bytes 等のプリミティブしか渡せない。Rust 側の構造体（termios, PtyState 等）を返すには不透明ハンドルが必要。Rust 側に `Vec<Option<T>>` を持ち、そのインデックスを Int で返すパターンを採用。close 時は `None` に置換し、インデックスは再利用しない。

```rust
// Rust 側
static PTY_TABLE: Mutex<Vec<Option<PtyState>>> = ...;

pub extern "C" fn shimux_pty_open(...) -> i32 {
    let state = PtyState { master_fd, slave_fd };
    let mut table = PTY_TABLE.lock().unwrap();
    let handle = table.len() as i32;
    table.push(Some(state));
    handle
}

pub extern "C" fn shimux_pty_close(handle: i32) -> i32 {
    let mut table = PTY_TABLE.lock().unwrap();
    // close 時は None に置換。インデックス再利用なし
    table[handle as usize] = None;
    0
}
```

```moonbit
// MoonBit 側
extern "C" fn shimux_pty_open(cols : Int, rows : Int) -> Int = "shimux_pty_open"
// 返り値は Int (ハンドル)。-1 で失敗。
```

TTY_TABLE も同一方針（`Vec<Option<TtyState>>`、close 時 None 置換、インデックス再利用なし）。

### DR: tty_size の複数値返却

ターミナルサイズ (rows, cols) を 1 回の ioctl で取得し、Int64 にパックして返す。

```
Int64 の下位 32bit に (rows << 16) | cols をパック。上位 32bit は 0。
失敗時は -1 (0xFFFFFFFFFFFFFFFF)。
MoonBit 側: rows = (result.lsr(16)).to_int(), cols = (result.land(0xFFFF_L)).to_int()
```

Design rationale: rows/cols を別関数にすると ioctl を 2 回呼ぶことになり DRY に反する。Bytes 出力引数は #borrow の考慮が複雑になるため、パッキングが最もシンプル。

### カテゴリ一覧

| カテゴリ | プレフィックス | 責務 |
|---|---|---|
| PTY | `shimux_pty_` | PTY ペア作成、子プロセス起動 |
| TTY | `shimux_tty_` | ターミナル属性制御 |
| I/O | `shimux_io_` | read/write/poll/close |
| Socket | `shimux_sock_` | Unix domain socket |
| Signal | `shimux_sig_` | シグナル制御 |
| Process | `shimux_proc_` | プロセス制御 |
| Daemon | `shimux_daemon_` | デーモン化 (Phase 4) |

### pty: PTY 操作

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_pty_open` | `cols: Int, rows: Int` | `Int` (handle) | PTY ペア作成。handle は内部テーブルのインデックス。-1 で失敗 |
| `shimux_pty_spawn` | `handle: Int, cmd: Bytes` | `Int` (child_pid) | handle の slave_fd で子プロセスを posix_spawn。SIGCHLD リセット + POSIX_SPAWN_SETSID + POSIX_SPAWN_CLOEXEC_DEFAULT (macOS) / close_range + CLOEXEC (Linux fallback) + 環境継承を内部処理。親の slave_fd も close。-1 で失敗 |
| `shimux_pty_master_fd` | `handle: Int` | `Int` (fd) | handle から master_fd を取得 |
| `shimux_pty_resize` | `handle: Int, cols: Int, rows: Int` | `Int` | master_fd に TIOCSWINSZ。0 で成功、-1 で失敗 |
| `shimux_pty_close` | `handle: Int` | `Int` | master_fd を close し、テーブルを None に置換。内部で WINCH_MASTER_FD を -1 にリセットする。0 で成功 |

DR: `pty_open` と `pty_spawn` を分離する理由 — テスタビリティ。PTY ペア作成と子プロセス起動を個別にテストできる。slave_fd は handle 内部で保持し、spawn 時に自動で使用・close するため、MoonBit 側が slave_fd を管理する必要はない。

DR: `pty_spawn` 内部で SIGCHLD を SIG_DFL にリセットする理由 — MoonBit ランタイムが SIGCHLD を横取りし waitpid が壊れるため（poc で発見）。

DR: fork ではなく posix_spawn を使う理由 — MoonBit の GC ランタイムと fork は非互換。子プロセスが GC 状態を継承して segfault する（poc で発見）。

DR: `pty_spawn` が `POSIX_SPAWN_CLOEXEC_DEFAULT` (macOS) を使う理由 — 明示的に開けた fd 以外を子プロセスで close し、fd 漏洩を防ぐ。Linux ではこのフラグが存在しないため、`close_range(3, UINT_MAX, CLOSE_RANGE_CLOEXEC)` (Linux 5.9+) または spawn 前の明示的 CLOEXEC 設定でフォールバックする。

### tty: ターミナル属性制御

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_tty_is_tty` | `fd: Int` | `Int` (0/1) | isatty ラッパー |
| `shimux_tty_size` | `fd: Int` | `Int64` | ターミナルサイズ。`(rows << 16) \| cols`。失敗時 -1 |
| `shimux_tty_save` | `fd: Int` | `Int` (handle) | 現在の termios と fd を保存しハンドルを返す。-1 で失敗 |
| `shimux_tty_set_raw` | `fd: Int` | `Int` | cfmakeraw + tcsetattr。0 で成功 |
| `shimux_tty_restore` | `handle: Int` | `Int` | 保存した termios を、保存時の fd に対して復元し、ハンドルを自動解放（None に置換）。0 で成功 |

対称ペア: `tty_save` ↔ `tty_restore`

DR: poc では `enable_raw_mode` / `disable_raw_mode` が内部 static Mutex で 1 つの termios を保持していたため、複数 fd に対応不可だった。ハンドル方式にすることで複数 fd の保存・復元が可能。

DR: `tty_save` が fd をハンドル内部に保持し、`tty_restore` は fd 引数不要とする理由 — save と restore で異なる fd を指定するのはバグの元。save 時の fd を内部保持することで不整合を防ぐ。

### io: I/O 操作

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_io_read` | `fd: Int, buf: Bytes, max_len: Int` | `Int` | read ラッパー。読み取りバイト数。0=EOF、-1=エラー（EAGAIN 含む） |
| `shimux_io_write` | `fd: Int, data: Bytes` | `Int` | ブロッキング fd に対して全バイト書き込み保証（partial write 時に内部リトライ）。ノンブロッキング fd では EAGAIN 時に -1 を返す（呼び出し側で poll 後にリトライ）。書き込みバイト数。-1 で失敗。Unix socket fd にも使用可能（write(2) による） |
| `shimux_io_close` | `fd: Int` | `Int` | close ラッパー |
| `shimux_io_set_nonblocking` | `fd: Int` | `Int` | O_NONBLOCK 設定 |
| `shimux_io_poll` | `fds: Bytes, nfds: Int, timeout_ms: Int` | `Int` | poll ラッパー。fds は packed pollfd 配列（エントリ 8B: fd:i32 + events:i16 + revents:i16、ネイティブエンディアン）。revents に結果を書き戻す。ready 数を返す。EINTR 時は内部リトライ |

DR: `io_write` のブロッキング/ノンブロッキング動作 — shimux のイベントループでは、write 対象の fd（stdout, master_fd）はブロッキングモードで使用する。read 対象の fd（stdin, master_fd, socket_fd）のみ `io_set_nonblocking` でノンブロッキング化する。したがって `io_write` は実質的に常にブロッキング write（全バイト保証）として動作する。EAGAIN 時 -1 の記述はノンブロッキング fd を渡された場合の防御的動作であり、通常のフローでは発生しない。

DR: `shimux_io_poll` の引数に Bytes で packed pollfd を渡す理由 — MoonBit FFI で配列やポインタの配列は渡せない。Bytes にバイナリパックすることで任意個数の fd を監視可能。MoonBit 側にパック/アンパックのヘルパーを用意する。

DR: `io_read` の -1 で EAGAIN と致命的エラーを区別しない理由 — shimux のイベントループは必ず `io_poll` で POLLIN を確認してから `io_read` を呼ぶ。EAGAIN は稀（spurious wakeup 等）であり、リトライで解消する。EIO 等の致命的エラー（PTY 切断 = shell 死亡）は `io_poll` の revents に POLLHUP/POLLERR として先に現れるため、read の戻り値ではなく poll の revents で fd のクローズを判断する。したがって `io_read` の -1 は一律「今回はデータなし、リトライ可能」として扱い、errno の区別は不要。

### sock: Unix domain socket

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_sock_listen` | `path: Bytes` | `Int` (fd) | socket + bind + listen。bind 前に umask(0o077) を設定し、bind 後に復元する（shimux はシングルスレッドのため安全。マルチスレッド化時は fchmod への移行が必要）。ソケットパスの親ディレクトリのパーミッションが 0700 であることを確認する。既存パスは unlink。-1 で失敗 |
| `shimux_sock_accept` | `fd: Int` | `Int` (client_fd) | accept ラッパー。EAGAIN 時は -1 |
| `shimux_sock_connect` | `path: Bytes` | `Int` (fd) | server に connect。-1 で失敗 |
| `shimux_sock_unlink` | `path: Bytes` | `Int` | ソケットファイルを unlink で削除。0 で成功、-1 で失敗 |

対称ペア: `sock_listen`(server) ↔ `sock_connect`(client)

ソケットの読み書きと close には `shimux_io_read` / `shimux_io_write` / `shimux_io_close` を使用する。Unix socket fd は通常の fd と同様に read(2)/write(2)/close(2) で操作可能なため、専用の send/recv/close は不要。

DR: sock カテゴリに close を持たない理由 — `io_close` で十分。sock_recv/sock_send を削除して io_read/io_write に統一したのと同じ理由で、close も io_close に統一する。sock カテゴリは「ソケット固有の操作」（listen, accept, connect, unlink）のみを担当する。

### sig: シグナル制御

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_sig_ignore` | `signum: Int` | `Int` | 指定シグナルを SIG_IGN に設定 |
| `shimux_sig_setup_winch` | `master_fd: Int` | `Int` | SIGWINCH ハンドラを設定し、受信時に master_fd へ TIOCSWINSZ 転送 |

`shimux_sig_reset_chld` は独立した FFI 関数としては公開しない。`shimux_pty_spawn` 内部で自動実行される。

Phase 4 では shimux-server 起動時に `sig_ignore(SIGHUP)` を呼び、クライアント切断時もシェルを保持する。

DR: `sig_setup_winch` が master_fd を引数に取る理由 — シグナルハンドラ内で master_fd を参照する必要があるため、Rust 側の AtomicI32 (WINCH_MASTER_FD) に保存する。MoonBit からハンドラ関数を渡すのは FFI の制約上困難。

注記: shimux-agent は 1 プロセスにつき 1 PTY を管理する。SIGWINCH ハンドラも 1 つの master_fd にのみ対応する。split-window 時は別の shimux-agent プロセスが起動する。

注記: SIGWINCH ハンドラ内では Atomic ロードと async-signal-safe syscall（ioctl, write 等）のみを使用すること。malloc, Mutex 操作等は禁止。

### proc: プロセス制御

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_proc_waitpid` | `pid: Int, nohang: Int` | `Int64` | waitpid ラッパー。`(status << 32) \| (result_pid & 0xFFFFFFFF)`。WNOHANG は nohang=1 で指定。失敗時は下位 32bit が `0xFFFFFFFF` |
| `shimux_proc_exit` | `status: Int` | `Unit` | プロセス終了 |
| `shimux_proc_kill` | `pid: Int, signum: Int` | `Int` | kill ラッパー |

DR: `proc_waitpid` が Int64 を返す理由 — pid と status の 2 値を返す必要があるため。失敗時は下位 32bit を 0xFFFFFFFF (unsigned) とし、MoonBit 側で `result.land(0xFFFFFFFF_L)` が 0xFFFFFFFF であれば失敗と判定する。

DR: `proc_exit` の存在理由 — MoonBit が C コードを生成する際に stdlib の `exit` シンボルと衝突するため、`shimux_proc_exit` として名前空間を分離する。

DR: `proc_waitpid` が `pid=-1`（任意の子プロセスを待つ）をサポートしない理由 — shimux は `pty_spawn` で起動した特定の子プロセスの pid を把握しており、常にその pid を指定して waitpid する。pid=-1 は MoonBit ランタイムの内部子プロセスまで刈り取ってしまうリスクがあるため、明示的な pid 指定のみサポートする。

DR: `proc_killpg` (pgid kill) を提供しない理由 — Claude Code のユースケースでは agent が起動するのは `$SHELL` であり、シェル自体がジョブ制御で子プロセスグループを管理する。shimux が pgid kill を行う必要はない。shimux は子シェルに対する `proc_kill` のみで十分。

### daemon: デーモン化 (Phase 4)

| 関数 | 引数 | 戻り値 | 説明 |
|---|---|---|---|
| `shimux_daemon_daemonize` | なし | `Int` | fork→親終了→setsid→/dev/null 付け替え。0 で成功 |
| `shimux_daemon_pid_write` | `path: Bytes` | `Int` | PID ファイル作成 |
| `shimux_daemon_pid_check` | `path: Bytes` | `Int` | 既存プロセスが生きているか確認。1=生存、0=不在、-1=エラー |
| `shimux_daemon_pid_remove` | `path: Bytes` | `Int` | PID ファイル削除 |

DR: fork + GC 非互換への対処 — `pty_spawn` で確認済みの通り、MoonBit の GC ランタイムと fork は非互換である。`daemon_daemonize` は fork を内包するため、直接実装すると GC ヒープ破損のリスクがある。Phase 4 実装時には以下の代替案から選択する:

1. **Rust 側 constructor で daemonize**: MoonBit ランタイム初期化前に double-fork + setsid を完了させる。MoonBit の main() はデーモン化済みプロセスで初めて実行される
2. **OS サービスマネージャに委任**: shimux 自体は foreground で動作し、launchd (macOS) / systemd (Linux) に daemon 管理を委ねる。`shimux_daemon_daemonize` FFI 自体が不要になる
3. **外部ラッパーバイナリ**: 極薄の Rust バイナリ `shimux-daemon` が daemonize 後に shimux-server を起動する

現時点では方式未決定。Phase 4 着手時に検証のうえ決定し、この DR を更新する。

### FFI 関数総数

| Phase | カテゴリ | 関数数 |
|---|---|---|
| Phase 1 | pty (5) + tty (5) + io (5) + sock (4) + sig (2) + proc (3) | **24** |
| Phase 4 | daemon (4) | **+4** |

変更履歴（レビュー指摘反映）:
- `sock_recv` 削除 → `io_read` で代替（Unix socket fd は read(2) で読める）
- `sock_send` 削除 → `io_write` で代替（同上）
- `sock_close(fd, path)` → `sock_close(fd)` + `sock_unlink(path)` に分離
- `sig_reset_chld` 削除 → `pty_spawn` 内部で自動実行
- `proc_killpg` 不採用 → DR 記載（シェルのジョブ制御に委任）
- `sock_close` 削除 → `io_close` で代替（Unix socket fd は close(2) で閉じられる）
- `daemon_daemonize` の fork 前提を見直し → fork/GC 非互換の代替案を DR に追記
- send-keys 経路を `observer.on_input()` 経由に変更 → 画面バッファ整合性のため

### 不要と判断した FFI（MoonBit core に既存 or 不要）

| poc2 の関数 | 不要理由 |
|---|---|
| `shimux_getenv` / `shimux_setenv` | `@env.get_env_var()` が MoonBit core にある |
| `shimux_errno` / `shimux_strerror` | エラーは各関数の戻り値で表現 |
| `shimux_monotonic_time_ms` | MoonBit core の `@time` で取得可能 |
| `shimux_splice` | ゼロコピーは初期実装で不要 |
| `shimux_epoll_*` / `shimux_kqueue_*` | `shimux_io_poll` に統一 |
| `shimux_log_*` | ログ I/O は `shimux_io_write` + MoonBit 側フォーマットで十分 |
| `shimux_signal_pipe_fd` | self-pipe トリックは poll の EINTR リトライで代替 |
| `shimux_sigmask_*` | posix_spawn 採用により fork 前後のマスク不要 |

## MoonBit 側インターフェース設計

### ハンドル型

```moonbit
// ハンドル種別の取り違えを型で防ぐ newtype ラッパー
type PtyHandle Int   // PTY ハンドル（内部テーブルのインデックス）
type TtyHandle Int   // TTY ハンドル（同上）
type SockFd Int      // ソケット fd（raw fd だがカテゴリ区別のためラップ）
```

FFI の戻り値 Int を即座にラップし、以降は `PtyHandle` / `TtyHandle` として扱う。異なるハンドル種別を誤って渡すとコンパイルエラーになる。

### エラーハンドリング方針

```moonbit
// Handle 系: -1 チェック後に Result でラップ
// 例: pty_open が -1 → Err(FFIError::PtyOpenFailed)
//     成功 → Ok(PtyHandle(handle))

// 0/-1 系: Result[Unit, FFIError] でラップ
// 例: pty_resize が -1 → Err(FFIError::ResizeFailed)

// Size 系: 呼び出し元で直接判定（イベントループ内で使うためラップのオーバーヘッドを避ける）
// 例: io_read の戻り値を直接 match で分岐
```

### 定数定義

```moonbit
// シグナル番号（macOS / Linux 共通）
// 注記: これらの値は macOS と Linux の両方で同一。
// プラットフォーム固有の値が必要な場合は FFI 経由で取得する。
let SIGHUP : Int = 1     // Phase 4: server が sig_ignore(SIGHUP) でクライアント切断を無視
let SIGPIPE : Int = 13
let SIGTERM : Int = 15
let SIGWINCH : Int = 28

// poll イベント
let POLLIN : Int = 0x0001
let POLLHUP : Int = 0x0010
let POLLERR : Int = 0x0008
```

### Observer trait

```moonbit
trait Observer {
  on_output(Self, Bytes) -> Bytes   // シェル出力を監視（tap）
  on_input(Self, Bytes) -> Bytes    // ユーザー入力を監視（tap）
  capture(Self) -> String           // 現在の画面内容を返す（capture-pane 用）
}

struct NullObserver {}  // Phase 3: 完全透過

impl Observer for NullObserver with
  on_output(_self, data) { data }
  on_input(_self, data) { data }
  capture(_self) { "" }
```

後に `ScreenObserver`（ANSI パーサ + 仮想画面バッファ）に差し替え。

### Pollfd ヘルパー

```moonbit
// packed pollfd: [fd:i32(4B), events:i16(2B), revents:i16(2B)] = 8B/entry
fn pollfd_pack(entries : Array[(Int, Int)]) -> Bytes { ... }
fn pollfd_revents(buf : Bytes, index : Int) -> Int { ... }
```

注記: ネイティブエンディアン (LE on x86-64/aarch64)。対象プラットフォーム: macOS/Linux の x86-64/aarch64 のみ。

## poc からの知見（こぼしてはいけないもの）

| 知見 | 新設計での反映先 |
|---|---|
| fork NG → posix_spawn | `shimux_pty_spawn` 内部で posix_spawn 使用。DR 記載 |
| SIGCHLD リセット必要 | `shimux_pty_spawn` 内部で自動実行 |
| String.to_bytes() は UTF-16 | MoonBit 側で `@utf8.encode()` を使う規約 |
| Bytes メモリレイアウト | Rust 側の `bytes_length()` ヘルパーで対応 |
| #borrow アノテーション | 全 Bytes 入力引数に付与 |
| C スタブ不要 | Rust extern "C" で直接リンク |
| POSIX_SPAWN_SETSID = 0x0400 | macOS 固有定数。libc クレートに未定義 |
| macOS sun_path 104B 制限 | セッション名 SHA-256 先頭 16 文字でハッシュ |

## Phase 計画

```
Phase 1: shimux-agent 基盤
    │     pty + tty + io + sock + sig + proc の FFI 実装
    │     MoonBit 側イベントループ
    │
    ├── Path A: 1 → 3 → 2a
    │   Phase 3: Observer null 実装（完全透過プロキシ）
    │   Phase 2a: agent 単体で完結するコマンド（send-keys, capture-pane, list-panes 等）
    │
    └── Path B: 1 → 4 → 2b
        Phase 4: server/client 分離 + daemon
        Phase 2b: セッション管理コマンド（new-session, split-window）
```

Phase 1 完了後、Path A と Path B は並行して進められる。
合流は自然にできる（observer は trait で抽象化、server/client は I/O 経路の変更のみ）。

合流後のコマンドルーティング:
- `send-keys`, `capture-pane`, `list-panes` → agent の Unix socket に直接 connect（Phase 2a のまま維持）
- `new-session`, `split-window`, `has-session` → server 経由（Phase 2b + Phase 4）
- server はセッション管理のみ担当し、ペイン内操作は agent に直接ルーティングする

注記: Phase 番号が時系列順でない理由 — Phase 2（tmux 互換 CLI）は Phase 3（Observer）と Phase 4（server/client）に依存するため、番号順ではなく依存グラフ順に進行する。番号は初期設計時の命名をそのまま維持している。

## ディレクトリ構成

```
poc3/
├── DESIGN.md              # この文書
├── moon.mod.json          # supported-targets: ["native"]
├── ffi/                   # Rust staticlib
│   ├── Cargo.toml
│   └── src/
│       ├── lib.rs         # モジュール宣言
│       ├── pty.rs         # shimux_pty_*
│       ├── tty.rs         # shimux_tty_*
│       ├── io.rs          # shimux_io_*
│       ├── sock.rs        # shimux_sock_*
│       ├── sig.rs         # shimux_sig_*
│       └── proc.rs        # shimux_proc_*
├── lib/
│   ├── ffi/               # extern "C" 宣言（カテゴリ別ファイル）
│   │   ├── moon.pkg.json  # cc-link-flags で ffi/target/release/libshimux_ffi.a をリンク
│   │   ├── pty.mbt
│   │   ├── tty.mbt
│   │   ├── io.mbt
│   │   ├── sock.mbt
│   │   ├── sig.mbt
│   │   └── proc.mbt
│   ├── agent/             # イベントループ、I/O プロキシ
│   │   ├── moon.pkg.json
│   │   └── agent.mbt
│   └── observer/          # Observer trait + 実装
│       ├── moon.pkg.json
│       ├── observer.mbt   # trait 定義
│       └── null.mbt       # NullObserver
├── cmd/
│   ├── agent/             # shimux-agent エントリーポイント
│   │   ├── moon.pkg.json
│   │   └── main.mbt
│   └── shimux/            # shimux CLI エントリーポイント (Phase 2)
│       ├── moon.pkg.json
│       └── main.mbt
└── lib/                   # Phase 2 以降
    ├── cli/               # tmux 互換引数解析 (Phase 2)
    └── command/           # コマンド変換層 (Phase 2)
```

Rust 側もカテゴリ別にモジュール分割し、MoonBit 側の ffi パッケージと 1:1 対応させる。

注記: `moon.mod.json` で `"supported-targets": ["native"]` を指定。`moon.pkg.json` で `"cc-link-flags"` に Rust staticlib のパスを指定する。
