# MoonBit + Rust FFI tty/pty PoC 知見

## 結論

MoonBit native backend + Rust FFI (C ABI) で tty/pty 制御は**実現可能**。tmux の透過的操作（send-keys, capture-pane, split-window 等）を含む全操作が動作確認済み。

## アーキテクチャ

```
MoonBit (native backend)
  └── extern "C" fn ... = "symbol_name"
        └── Rust staticlib (crate-type = ["staticlib"])
              └── libc クレート → POSIX API
```

## 知見一覧

### 1. リンク方法

- C スタブ（.c ファイル）不要。Rust `extern "C"` で直接 C ABI 公開
- `moon.pkg.json` の `link.native.cc-link-flags` に `-L<path> -l<name>` を指定するだけ
- `native-stub` 設定も不要

```json
{
  "link": {
    "native": {
      "cc-link-flags": "-Lffi/target/release -lshimux_ffi"
    }
  }
}
```

### 2. fork NG、posix_spawn OK

MoonBit の GC ランタイムと `fork()` は非互換。子プロセスが GC 状態を継承して segfault する。

**回避策**: `openpty()` + `posix_spawn()` を使う。posix_spawn は fork 後にユーザーコードを実行しないため安全。

macOS では `POSIX_SPAWN_SETSID` (0x0400) で新セッション作成（`setsid()` 相当）。libc クレートにはまだ定数がない。

### 3. SIGCHLD リセット必要

MoonBit ランタイムが SIGCHLD ハンドラを設定して子プロセスを自動 reap するため、`waitpid()` が常に ECHILD で失敗する。

**回避策**: 子プロセス生成前に `sigaction(SIGCHLD, SIG_DFL)` でリセット。

### 4. String.to_bytes() は UTF-16

MoonBit の `String.to_bytes()` は **UTF-16 LE** を返す（ASCII 文字間にヌルバイトが入る）。C/Rust 境界では使えない。

**正しい方法**: `@utf8.encode(s)` で UTF-8 Bytes を取得。パッケージ依存に `moonbitlang/core/encoding/utf8` を追加。

### 5. Bytes のメモリレイアウト

MoonBit Bytes は C 側で `uint8_t*`（データ先頭へのポインタ）として渡される。ヘッダはデータの直前:

```
[rc: i32] [meta: u32] [data: u8...]
                       ^--- ポインタはここを指す
```

- `meta` の下位 28bit がバイト長
- Rust 側から新しい Bytes を作るには `moonbit_make_bytes_raw(size)` を呼ぶ

### 6. #borrow アノテーション

Rust 側が入力データの所有権を取らない場合、MoonBit 側で `#borrow(param)` を付ける。GC が不要な decref を行わないようになる。

### 7. I/O プロキシの実装

- `poll()` で stdin と PTY master fd を同時監視
- SIGWINCH は `sigaction` でハンドラ登録、`TIOCSWINSZ` で PTY に転送
- raw mode では `\n` だけだと CR が入らない → `\r\n` を明示的に write

### 8. supported-targets 設定

`extern "C"` は native 限定なので、`moon.pkg.json` に `"supported-targets": ["native"]` を設定。`moon check` のデフォルト (wasm-gc) でエラーにならない。

## mizchi の MoonBit 関連調査

- **Luna.mbt**: Web UI フレームワーク（ターミナル TUI ではない）
- **libgit2 バインディング**: C FFI で約40関数をバインド。C FFI の実用性を実証
- **tui-poc**: Rust (Ratatui) 製。MoonBit ではない
- **結論**: MoonBit で低レイヤー tty/pty を直接扱う先行事例はない。Rust FFI が現実的アプローチ

### 9. cc-link-flags と moon build の挙動（native target 特有の制約）

`cc-link-flags` を持つ lib パッケージは、`moon build` 時に単体で exe リンクの対象になる。native target では実行可能バイナリが出力されるため、リンカが `_main` エントリポイントを要求するが、lib パッケージには `main()` がないためリンクエラーになる。

**wasm-gc / js target では発生しない**: 出力がモジュール/バンドルでありエントリポイントが不要なため。`cc-link-flags` 自体が `link.native` 配下の設定なので native target 限定の問題。

**再現確認**: mizchi/tui.mbt でも同じ現象を確認済み。`src/io/moon.pkg` に C ファイル直接指定の `cc-link-flags` があり、`moon build --target native` で `_main` 未定義エラーが発生する。C ファイル直接指定か `-L -l` 形式かは無関係。

| コマンド | lib に cc-link-flags あり | 結果 |
|---|---|---|
| `moon check` | 影響なし | 成功 |
| `moon test` | テストドライバが main() を自動生成 | 成功 |
| `moon build`（パッケージ未指定） | lib パッケージも exe 生成対象 | `_main` 未定義エラー |
| `moon build cmd/agent` | main パッケージのみビルド | 成功 |

**回避策**: ビルドコマンドで `is-main: true` のパッケージを明示指定する。

```bash
# NG
moon build --target native

# OK
moon build cmd/agent --target native
```

**lib パッケージに cc-link-flags が必要な理由**: `moon test` は自動的にテストランナー（main あり）を生成する。lib/ffi のテスト（`_wbtest.mbt`）が直接 FFI 関数を呼ぶため、リンクフラグなしではテストが動かない。

## poc (初期PoC) の構成

```
poc/
├── moon.mod.json
├── cmd/main/
│   ├── moon.pkg.json    # native限定, Rust .a リンク
│   └── main.mbt         # extern "C" 宣言 + main + テスト (18件)
└── ffi/
    ├── Cargo.toml        # staticlib, libc依存
    └── src/lib.rs        # FFI 関数群
```

### FFI 関数一覧 (poc)

| 関数 | 用途 |
|---|---|
| `shimux_add` | PoC: Int 受け渡し |
| `shimux_to_upper` | PoC: Bytes 受け渡し |
| `shimux_bytes_len` | PoC: Bytes 長取得 |
| `shimux_isatty` | TTY 判定 |
| `shimux_get_rows` / `shimux_get_cols` | ターミナルサイズ |
| `shimux_enable_raw_mode` / `shimux_disable_raw_mode` | raw mode 制御 |
| `shimux_read_byte` | 1バイト読み取り |
| `shimux_write` | fd への Bytes 書き込み |
| `shimux_forkpty_exec` | PTY + 子プロセス起動 |
| `shimux_proxy_io` | 双方向 I/O プロキシ + ログ |
| `shimux_exit` | プロセス終了 |

### 使い方

```bash
# ビルド
(cd poc/ffi && cargo build --release)
(cd poc && moon build --target native)

# テスト
(cd poc && moon test --target native)

# bash を PTY プロキシ経由で起動（ログ付き）
(cd poc && moon run cmd/main --target native -- bash /tmp/shimux.log)

# tmux を PTY プロキシ経由で起動
(cd poc && moon run cmd/main --target native -- tmux /tmp/shimux-tmux.log)
```

## poc3 実装状況

Phase 1 FFI 層の実装とレビューが完了。

- Rust FFI: 24関数（pty/tty/io/sock/sig/proc 6カテゴリ）
- MoonBit バインディング: 型安全ラッパー（PtyHandle/TtyHandle/SockFd newtype）
- Observer trait + NullObserver（完全透過）
- テスト: 27件全パス、moon check 警告 0
- 2ラウンドのレビュー＋修正完了（Critical 0、Warning 0）

詳細: `poc3/` ディレクトリ
