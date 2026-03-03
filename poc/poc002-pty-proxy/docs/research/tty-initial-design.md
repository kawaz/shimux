# shimux 設計会話トランスクリプト

---

## MoonBitでTTYを扱うコードを書きたい

**kawaz:** moonbitでttyを扱うコードを書きたいと思ってる。io部分はFFIで解決する必要があると思うがどう言う構造にするのが最適そうでしょう？

**Claude:**

### MoonBitでTTYを扱う構造

まずターゲット選択が設計を大きく左右します。

#### ターゲット別の現実

| ターゲット | Raw mode | カーソル制御 | 現実的か |
|---|---|---|---|
| JS (Node.js) | `process.stdin.setRawMode()` | ANSIエスケープ | ✅ 最も実用的 |
| Wasm (WASI) | **preview1にAPIなし** | fd_writeにエスケープ流すだけ | ⚠️ 制限大 |
| Native | C FFIでtermios | 可能 | 🚧 まだ不安定 |

Node.jsターゲットが現状最も現実的です。

#### 推奨構造

**traitで抽象化して、バックエンドをFFI境界に閉じ込める**

```moonbit
// コアのtrait定義（pure MoonBit）
trait TTYBackend {
  read_byte(Self) -> Int        // -1 = EOF
  write_bytes(Self, Bytes) -> Unit
  set_raw_mode(Self, Bool) -> Unit
  get_size(Self) -> (Int, Int)  // (cols, rows)
}

// 上位ロジックはtraitだけに依存
struct TTY[B : TTYBackend] {
  backend : B
  buf : Buffer
}
```

#### JS (Node.js) バックエンド

FFIを薄いglue層に徹する：

```moonbit
// ffi/node_tty.mbt
extern "js" fn ffi_set_raw_mode(enabled : Bool) -> Unit =
  #| (enabled) => process.stdin.setRawMode(enabled)

extern "js" fn ffi_read_byte() -> Int =
  #| () => {
  #|   const buf = Buffer.alloc(1);
  #|   const n = require('fs').readSync(0, buf, 0, 1);
  #|   return n === 0 ? -1 : buf[0];
  #| }

extern "js" fn ffi_write(bytes : Bytes) -> Unit =
  #| (bytes) => process.stdout.write(Buffer.from(bytes))

extern "js" fn ffi_get_size() -> Array[Int] =
  #| () => [process.stdout.columns, process.stdout.rows]

// MoonBit側のwrapper
struct NodeTTYBackend {}

impl TTYBackend for NodeTTYBackend with
  read_byte(_self) { ffi_read_byte() }
  write_bytes(_self, b) { ffi_write(b) }
  set_raw_mode(_self, e) { ffi_set_raw_mode(e) }
  get_size(_self) { 
    let s = ffi_get_size()
    (s[0], s[1])
  }
```

#### ANSIエスケープはpure MoonBitで

ここはFFI不要なのでMoonBitで完結させる：

```moonbit
// ansi.mbt - FFI依存ゼロ
fn cursor_move(col : Int, row : Int) -> Bytes {
  "\x1b[\{row};\{col}H".to_bytes()
}

fn clear_screen() -> Bytes {
  "\x1b[2J\x1b[H".to_bytes()
}

fn set_fg_color(r : Int, g : Int, b : Int) -> Bytes {
  "\x1b[38;2;\{r};\{g};\{b}m".to_bytes()
}
```

#### 全体の依存方向

```
[pure MoonBit]
  ANSIエスケープ生成
  キーイベントパース（\x1b[Aなど）
  アプリケーションロジック
        ↓ TTYBackend trait
[FFI境界]
  NodeTTYBackend / WasiTTYBackend
        ↓
[外部]
  Node.js process.stdin/stdout
  WASI fd_read/fd_write
```

---

## shimux（tmux的なもの）のアイデア

**kawaz:** やりたい事としては、tmuxのように、例えばghosttyの中でshimux(アプリ名仮称)を起動したら内部で$SHELLを起動して傍目には何も変化がないように使えるが、ghosttyが再起動されてもshimux-serverが保持してるpty？は死なずその上で$SHELLは生きていて、再起動後のghosttyからshimux attachしたらさっきのシェルが引き続き使えるみたいな奴です。そしてPTYのIOを基本は透過しつつログを取ったりフックしたりしたい。PoCとしてはまずコレを実現したい。

**Claude:**

これはほぼtmuxと同じ問題空間ですね。技術的に整理します。

### コアとなる構造

```
[Ghostty]
    ↕ TTY (stdin/stdout)
[shimux-client]
    ↕ Unix domain socket
[shimux-server (daemon)]
    ↕ PTY master fd
[PTY slave ↔ $SHELL]
```

Ghosttyが死んでもsocket越しの接続が切れるだけで、**server↔shell間のPTYは生き続ける**のがポイント。

### PTYの仕組み（ここがコア）

```
openpty() → master_fd / slave_fd のペアを取得
fork()
  子プロセス: slave_fdを stdin/stdout/stderr にして exec($SHELL)
  親プロセス: master_fdを持ち続ける ← shimux-serverがここ

master_fdへのread/write = shellのI/O
IOCTLでウィンドウサイズ変更(TIOCSWINSZ)も master_fd 経由
```

### MoonBitでの現実的なPoCターゲット

**Node.js（PoC最速）**

```moonbit
extern "js" fn ffi_pty_spawn(shell : String) -> Int =
  #| (shell) => {
  #|   const pty = require('node-pty');
  #|   const p = pty.spawn(shell, [], {
  #|     name: 'xterm-256color',
  #|     cols: 80, rows: 24,
  #|     env: process.env
  #|   });
  #|   globalThis.__pty_map = globalThis.__pty_map || {};
  #|   const id = Date.now();
  #|   globalThis.__pty_map[id] = p;
  #|   return id;
  #| }
```

### 全体のコンポーネント設計

```moonbit
struct Session {
  id : String
  pty_id : Int
  log_buf : Buffer
  hooks : Array[Hook]
}

struct ShimuxServer {
  sessions : Map[String, Session]
  socket_path : String
}

enum ClientMessage {
  Attach(session_id : String)
  Detach
  Input(data : Bytes)
  Resize(cols : Int, rows : Int)
  NewSession
}

enum ServerMessage {
  Output(data : Bytes)
  SessionList(Array[SessionInfo])
  Attached(session_id : String)
}

enum HookResult {
  Passthrough
  Replace(Bytes)
  Suppress
  Tap(Bytes, Bytes)
}
```

### I/O透過とフックの流れ

```
[shell output]
  → master_fd から read
  → Hook chain を通す（ログ取り、パターン検知など）
  → 接続中の全clientに broadcast
  → client → Ghosttyのstdout に write

[Ghostty input]
  → client が stdin から read
  → サーバーに転送
  → Hook chain
  → master_fd に write → shellに届く
```

### PoCの実装順序

1. node-ptyでPTYをFFIラップ → shellが起動してI/Oが通ることを確認
2. Unix socketでserver/client分離 → クライアントを切ってもshellが死なないことを確認
3. clientのTTYサイズをserverに伝えてTIOCSWINSZ → vimなどが正しく動くことを確認
4. Hookチェーン実装 → 出力ログ取得

---

## openpty/fork/exec/setsidの難しさ

**kawaz:** イメージは掴めてきた。ちなみにopenpty, fork, exec, setsidまでやると大変とのことですがどのあたりが難しい? また、やることはコレで全てですか?

**Claude:**

### 難しい部分

**fork()後の処理が繊細**

```c
pid = fork()
// 子プロセス側でやること（順番も重要）
setsid()
close(master_fd)
ioctl(slave_fd, TIOCSCTTY, 0)
dup2(slave_fd, 0)  // stdin
dup2(slave_fd, 1)  // stdout  
dup2(slave_fd, 2)  // stderr
close(slave_fd)
execve($SHELL)
```

順番を間違えると「shellは起動してるがTTYとして認識されない」など気づきにくいバグになる。

**シグナル処理**

```
SIGCHLD  - shellが死んだことを検知
SIGWINCH - ウィンドウサイズ変更をmaster_fdにIOCTLで伝える
SIGHUP   - クライアント切断時にshellを殺さないよう無視
```

**Non-blocking I/O + イベントループ**

```
master_fdはblocking readしてると他の処理が詰まる
→ epoll(Linux) / kqueue(macOS) でイベント駆動にする必要がある
→ Unix socketも同じループで監視
```

### 全体のやることリスト

```
[Server]
□ daemonize
□ PIDファイル管理
□ openpty
□ fork + setsid + dup2 + exec($SHELL)
□ SIGCHLD ハンドラ（waitpid）
□ SIGHUP 無視
□ epoll/kqueuでmaster_fd + unix socketを同時監視
□ クライアント管理（複数attach想定）
□ TIOCSWINSZ（クライアントからのリサイズ反映）
□ スクロールバックバッファ（任意）
□ Hookチェーン

[Client]
□ Unix socketでserverに接続
□ stdinをRAW modeに設定
□ 終了時にTTY restore（atexit/シグナル）
□ SIGWINCH → serverにサイズ変更通知
□ stdin → server I/O転送
□ server出力 → stdoutへ転送
□ SIGINT等のシグナルをmaster_fdに正しく伝える
```

---

## Native = C FFI → Rustの方が良い

**kawaz:** nativeでいけちゃうの？

**Claude:** 正直なところ、MoonBitのNativeターゲットの現状の安定度について正確な情報を持っていません。確認すべき点は、C FFIが実用レベルで動くか、ポインタ・構造体をFFI越しに扱えるか、などです。

**kawaz:** あーネイティブ=CFFIてことか。

**Claude:** そうです。MoonBitの「Nativeターゲット」は今のところ**「CにコンパイルしてGCCでビルド」**という構造なので、

```
MoonBit → C → gcc/clang → ネイティブバイナリ
```

つまりNativeターゲットで`extern "c"`でFFI書く = Cの関数を直接呼ぶ、という意味です。最悪でも「PTY周りだけ薄いCのwrapper関数を書いてMoonBitから呼ぶ」という逃げ道もあります。

```c
// pty_wrapper.c
int shimux_openpty(int *master, int *slave) {
    return openpty(master, slave, NULL, NULL, NULL);
}
```

```moonbit
extern "c" fn shimux_openpty(master : Ref[Int], slave : Ref[Int]) -> Int = "shimux_openpty"
```

**kawaz:** なるほどなるほど。とするとネイティブFFIの先はRustの方が良いな。Cより

**Claude:**

それは良い選択です。

**PTY周りのcrateが充実している**

```toml
[dependencies]
nix = { features = ["pty", "process", "signal"] }
tokio = { features = ["full"] }
```

**C ABI互換なのでMoonBitから呼べる**

```rust
#[no_mangle]
pub extern "C" fn shimux_openpty() -> i32 { ... }
```

```moonbit
extern "c" fn shimux_openpty() -> Int = "shimux_openpty"
```

**結果的な構造**

```
[MoonBit]
  セッション管理、プロトコル、Hookロジック
        ↓ C ABI
[Rust (cdylib or staticlib)]
  PTY, fork/exec, epoll/kqueue, signal handling
        ↓
[OS]
```

---

## 必要なFFI 網羅リスト

**kawaz:** 必要なFFIを網羅的にエッジケースの対応も含めて全て列挙して。

**Claude:**

### 1. デーモン化

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_daemonize` | fork→親終了→setsid→/dev/null付け替え | 既にデーモンの場合の二重実行防止 |
| `shimux_pid_file_write(path)` | PIDファイル作成 | ファイルが既存＝前回クラッシュ、PIDが生きてるか確認が必要 |
| `shimux_pid_file_check(path)` | 既存プロセスが生きてるか確認 | staleなPIDファイルの処理 |
| `shimux_pid_file_remove(path)` | 終了時にPIDファイル削除 | シグナル終了時も確実に呼ばれるようatexit登録 |

### 2. PTY操作

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_pty_open(cols, rows)` → `PtyHandle` | openptyでmaster/slaveペア取得 | 失敗時のfd leak防止 |
| `shimux_pty_master_fd(handle)` → `Int` | master fdを取得 | - |
| `shimux_pty_slave_fd(handle)` → `Int` | slave fdを取得 | exec後は子プロセスのみが持つべき |
| `shimux_pty_close_slave(handle)` | 親プロセスでslave_fdを閉じる | forkの直後に必須、忘れるとshell終了を検知できない |
| `shimux_pty_resize(master_fd, cols, rows)` | TIOCSWINSZ | shellがSIGWINCHを受け取れるか確認 |
| `shimux_pty_get_size(master_fd)` → `(cols, rows)` | TIOCGWINSZ | - |
| `shimux_pty_set_nonblocking(master_fd)` | O_NONBLOCKを設定 | epollと組み合わせ必須 |

### 3. プロセス管理

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_spawn(slave_fd, shell, env[])` → `Pid` | fork+setsid+dup2+execve | execve失敗時に子プロセスが_exit()すること |
| `shimux_waitpid_nonblock(pid)` → `WaitResult` | ゾンビ回避 | EINTR対応、WNOHANG必須 |
| `shimux_kill(pid, signal)` | shellへのシグナル送信 | pgid単位で送るべきケース |
| `shimux_getpgid(pid)` → `Int` | プロセスグループ取得 | - |
| `shimux_killpg(pgid, signal)` | プロセスグループ全体にシグナル | shellが子プロセスを持つ場合 |

**spawn内部の子プロセス側処理順（順番厳守）：**

```
1. setsid()
2. close(master_fd)
3. ioctl(slave_fd, TIOCSCTTY, 0)
4. dup2(slave_fd, 0/1/2)
5. close(slave_fd)
6. 環境変数設定（TERM=xterm-256color等）
7. execve($SHELL)
8. 失敗したら_exit(1)
```

### 4. シグナル処理

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_signal_ignore(signum)` | SIGHUP等を無視 | serverはSIGHUPを無視してclientが死んでもshellを保持 |
| `shimux_signal_setup_chld(callback)` | SIGCHLD→waitpid | signal handlerからはwrite-to-pipeのみ安全 |
| `shimux_signal_setup_winch(callback)` | client側SIGWINCH検知 | - |
| `shimux_signal_setup_term(callback)` | SIGTERM/SIGINT終了処理 | TTY restoreとPIDファイル削除を確実に |
| `shimux_signal_pipe_fd()` → `Int` | self-pipeトリック用fd | signal handlerからepollループに通知 |
| `shimux_sigmask_block_all()` | fork直前にシグナルを全マスク | fork中にシグナルが来ると子プロセスの状態が壊れる |
| `shimux_sigmask_restore()` | fork後にマスク解除 | - |

### 5. Unix Domain Socket

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_server_socket_create(path)` → `Fd` | SOCKSTREAMソケット作成+bind+listen | pathが既存の場合unlink→再作成 |
| `shimux_server_socket_accept(fd)` → `Fd` | accept | EAGAIN/EINTRで再試行 |
| `shimux_client_socket_connect(path)` → `Fd` | serverに接続 | serverが未起動の場合のauto-start |
| `shimux_socket_close(fd)` | ソケットクローズ | server終了時にunlink(path)も必要 |
| `shimux_socket_set_nonblocking(fd)` | O_NONBLOCK | epoll必須 |
| `shimux_socket_send(fd, bytes, len)` → `Int` | データ送信 | EAGAIN/EPIPEの処理 |
| `shimux_socket_recv(fd, buf, len)` → `Int` | データ受信 | 0バイト=client切断 |
| `shimux_socket_send_all(fd, bytes, len)` | 全バイト送信保証 | send()は部分送信することがある |
| `shimux_socket_cleanup(path)` | 終了時にsocketファイル削除 | - |

### 6. epoll/kqueuイベントループ

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_epoll_create()` → `Fd` | epoll fd作成 | Linux専用、macOSはkqueue |
| `shimux_epoll_add(epfd, fd, events)` | fdを監視対象に追加 | EPOLLET/EPOLLLTの選択 |
| `shimux_epoll_mod(epfd, fd, events)` | 監視イベント変更 | 書き込み可能になったら削除するパターン |
| `shimux_epoll_del(epfd, fd)` | 監視解除 | fd close前に必ずdel |
| `shimux_epoll_wait(epfd, events[], max, timeout_ms)` → `Int` | イベント待機 | EINTR対応必須 |
| `shimux_kqueue_create()` → `Fd` | macOS用 | - |
| `shimux_kqueue_add(kqfd, fd, filter)` | 監視追加 | EVFILT_READ/EVFILT_WRITE |
| `shimux_kqueue_wait(kqfd, events[], max, timeout_ms)` → `Int` | イベント待機 | - |

実用上はRust側でtokioを使えばepoll/kqueuを直接触らなくてよくなります。

### 7. クライアント側TTY制御

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_tty_save(fd)` → `TermiosHandle` | 現在のtermios保存 | attach前に必ず保存 |
| `shimux_tty_set_raw(fd)` | raw mode設定 | ECHO/ICANON/ISIG等をオフ |
| `shimux_tty_restore(fd, handle)` | termios復元 | SIGTERM/SIGINT/exit全パスで必ず呼ぶ |
| `shimux_tty_is_tty(fd)` → `Bool` | isatty()確認 | パイプ経由起動時の判定 |
| `shimux_tty_get_size(fd)` → `(cols, rows)` | 現在の端末サイズ取得 | SIGWINCH受信時にも呼ぶ |

### 8. I/O操作

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_read_nonblock(fd, buf, len)` → `Int` | ノンブロッキングread | EAGAIN=データなし、EIO=PTY切断（shellが死んだ） |
| `shimux_write_all(fd, buf, len)` → `Int` | 全バイト書き込み保証 | 部分書き込み対応、EPIPE=相手が死んでいる |
| `shimux_splice(src_fd, dst_fd)` | fd間の直接コピー | Linuxのみ、ゼロコピー転送 |
| `shimux_set_cloexec(fd)` | FD_CLOEXECを設定 | exec後にfdが子プロセスに漏れないよう全fdに設定 |

### 9. ログ・ファイルIO

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_log_open(path)` → `Handle` | ログファイルオープン | O_APPEND必須 |
| `shimux_log_write(handle, bytes, len)` | ログ書き込み | バッファリングせずwrite()直接 |
| `shimux_log_close(handle)` | クローズ | - |
| `shimux_log_rotate(handle, path)` | ログローテート | rename→新規作成のアトミック操作 |

### 10. 雑多・ユーティリティ

| 関数名 | 用途 | エッジケース |
|---|---|---|
| `shimux_getenv(name)` → `String` | 環境変数取得 | NULLチェック必須 |
| `shimux_setenv(name, val)` | 環境変数設定 | exec前のTERM等の設定用 |
| `shimux_errno()` → `Int` | 最後のエラー取得 | スレッドセーフなerrno取得 |
| `shimux_strerror(errno)` → `String` | エラー文字列変換 | - |
| `shimux_monotonic_time_ms()` → `Int64` | 単調増加時刻 | タイムアウト計算用 |

### まとめ：MoonBit側が管理するもの

上記FFIを全部Rust側に閉じ込めると、MoonBitは以下だけを扱えばよくなります：

- セッションの状態管理（`Map[String, Session]`）
- client/server間プロトコルのシリアライズ
- Hookチェーンのロジック
- スクロールバックバッファの管理
- CLIコマンドのパース
