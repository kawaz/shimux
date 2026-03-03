# tty-overlay PoC 実装計画

## Context

shimux の PTY プロキシ知見（poc002-pty-proxy）を活かし、子プロセスの出力上にテキストオーバーレイを常時表示する `tty-overlay` コマンドの PoC を作る。

配置先: `poc/poc003-tty-overlay/`

## CLI 仕様

```
tty-overlay [options] -- command [args...]
```

| オプション | 説明 | デフォルト |
|---|---|---|
| `--content TEXT` | 静的テキスト | (content/content-command のいずれか必須) |
| `--content-command CMD` | `bash -c CMD` の出力をテキストに | |
| `--interval DURATION` | content-command の更新間隔 | `5s` |
| `--width N\|auto` | オーバーレイ幅 | `auto` |
| `--height N\|auto` | オーバーレイ高さ | `auto` |
| `--top N` / `--bottom N` | 縦位置（排他） | `--top 1` |
| `--left N` / `--right N` | 横位置（排他） | `--right 1` |
| `--fg-color COLOR\|auto` | 文字色 | `auto` |
| `--bg-color COLOR\|auto` | 背景色 | `auto` |

## アーキテクチャ

```
stdin ──→ [Runner: poll loop] ──→ PTY master ──→ 子プロセス
                ↓
PTY master out ──→ stdout + オーバーレイ描画(ANSI escape)
```

オーバーレイ描画: 子プロセス出力の後に `DECSC → CUP → SGR+テキスト → DECRC` を注入。

## ファイル構成

```
poc/poc003-tty-overlay/
  PLAN.md
  moon.mod.json
  .gitignore
  ffi/                        # Rust FFI (poc002から流用 + exec.rs 新規)
    Cargo.toml
    src/
      lib.rs                  # 流用 + shimux_argc/argv/monotonic_ms 追加
      pty.rs, tty.rs, io.rs, sig.rs, proc.rs  # 流用
      exec.rs                 # [新規] posix_spawn+pipe でコマンド出力キャプチャ
  lib/
    ffi/                      # MoonBit FFI バインディング (流用 + exec.mbt 追加)
    overlay/                  # オーバーレイ描画ロジック
      types.mbt               # OverlayState
      render.mbt              # ANSI escape 生成
    args/                     # CLI 引数パーサ（簡易版）
      args.mbt, types.mbt
    runner/                   # イベントループ
      runner.mbt
  cmd/
    main/
      main.mbt
```

## FFI 流用・追加

### 流用（poc002/poc3/ffi/src/ からコピー）
- `lib.rs`, `pty.rs`, `tty.rs`, `io.rs`, `sig.rs`, `proc.rs`
- sock.rs は **不要**（ソケット通信なし）

### 新規追加

1. **exec.rs** — `shimux_exec_capture(cmd) -> Bytes`
   - posix_spawn + pipe で `/bin/sh -c cmd` を実行、stdout をキャプチャ
   - fork 回避（MoonBit GC との非互換のため、pty_spawn と同じパターン）

2. **pty.rs 追加** — `shimux_pty_spawn_argv(handle, argc, argv) -> pid`
   - 既存 `pty_spawn` は `/bin/sh -c cmd` 固定で、空白・メタ文字を含む引数が壊れる
   - `-- command [args...]` を安全に execv するため、argv 配列を直接受け取る版を追加

3. **lib.rs 追加** — `shimux_argc()`, `shimux_argv(i)`, `shimux_monotonic_ms()`
   - CLI 引数取得（MoonBit native にはビルトインがないため）
   - モノトニック時刻（content-command のタイマー制御用。後述）

### MoonBit FFI バインディング
- poc002 の `lib/ffi/*.mbt` から sock.mbt 除外でコピー
- `exec.mbt`, `helpers.mbt` に追加分のバインディング

## 主要型設計

### Config（args）
```moonbit
pub(all) enum ContentSource { Static(String) | Command(String) }
pub(all) enum VerticalPos { Top(Int) | Bottom(Int) }
pub(all) enum HorizontalPos { Left(Int) | Right(Int) }
pub(all) struct Config {
  content_source : ContentSource
  interval_ms : Int          // デフォルト 5000
  width : Int?               // None = auto
  height : Int?              // None = auto
  vertical : VerticalPos     // デフォルト Top(1)
  horizontal : HorizontalPos // デフォルト Right(1)
  fg_color : String?         // None = auto
  bg_color : String?         // None = auto
  command : Array[String]    // -- 以降
}
```

### OverlayState（overlay）
```moonbit
pub struct OverlayState {
  mut text : String
  mut lines : Array[String]
  mut width : Int
  mut height : Int
  mut term_rows : Int
  mut term_cols : Int
  config_width : Int?        // None = auto
  config_height : Int?
  vertical : VerticalPos
  horizontal : HorizontalPos
  fg_color : String?
  bg_color : String?
}
```

### Runner（runner）
```moonbit
pub struct Runner {
  pty_handle : @ffi.PtyHandle
  master_fd : Int
  child_pid : Int
  tty_handle : @ffi.TtyHandle?
  overlay : @overlay.OverlayState
  config : @args.Config
  mut stdin_open : Bool
}
```

## イベントループ（Runner::poll_once）

poc002 の Agent::poll_once パターンを踏襲。ソケット監視を除去し、タイマー + オーバーレイ注入を追加。

### タイマー設計（codex指摘対応）

poll timeout だけに依存すると、子プロセスが出力し続ける間タイムアウトが発火せず content-command が更新されない。
→ `shimux_monotonic_ms()` でモノトニック時刻を取得し、次回更新時刻を管理。poll_once 先頭で時刻チェックする。

### リサイズ伝播（codex指摘対応）

既存 `sig_setup_winch(master_fd)` が SIGWINCH で子PTYサイズを自動同期する（pty.rs の ioctl TIOCSWINSZ）。
加えて poll_once 先頭で `tty_size(0)` を確認し、変化時に `overlay.update_size()` でオーバーレイ位置を再計算。

### オーバーレイ注入タイミング

子プロセス出力のチャンク境界は ANSI シーケンス/UTF-8 の途中に来る可能性がある。
対策: stdout への write を `子プロセス出力 + オーバーレイ` を **単一の write 呼び出し**で結合して書き出す。
これにより ANSI シーケンスの途中にオーバーレイが割り込むことを防ぐ。

```
1. tty_size(0) で端末サイズ変更チェック → overlay.update_size()
2. content-command 使用時: monotonic_ms で次回更新時刻を過ぎていたら refresh
3. poll timeout 計算: static→-1, command→次回更新までの残りms
4. poll(stdin, master_fd)
5. stdin POLLIN: read → write to master (パススルー)
6. master POLLIN: read → 子出力 + overlay.render() を結合して単一 write で stdout へ
7. master POLLHUP: return false（子プロセス終了）
```

## 実装フェーズ

| Phase | 内容 |
|---|---|
| 0 | プロジェクト雛形 + FFI 移植（sock除外）+ argc/argv 追加 + ビルド確認 |
| 1 | exec.rs 実装 + テスト |
| 2 | args パーサ + テスト |
| 3 | overlay 描画ロジック + テスト（ANSI escape スナップショット） |
| 4 | Runner イベントループ + cmd/main + 統合テスト |
| 5 | 位置・色オプション + エッジケース + --help |

## 検証方法

```bash
# ビルド
(cd poc/poc003-tty-overlay/ffi && cargo build --release)
(cd poc/poc003-tty-overlay && moon build cmd/main --target native)

# テスト
(cd poc/poc003-tty-overlay && moon test --target native)

# 手動検証
./tty-overlay --content "Hello World" -- bash
./tty-overlay --content-command "date +%H:%M:%S" --interval 2s -- bash
./tty-overlay --content "Bottom Left" --bottom 1 --left 1 -- bash
./tty-overlay --content "Red on Cyan" --fg-color red --bg-color cyan -- bash
```

## 既知の制約（PoC として受容）

- **描画ちらつき**: 高速出力時。緩和策は Synchronized Output Mode（PoC 範囲外）
- **マルチバイト幅**: 全角文字で幅ずれ。codepoint 数で代用
- **DECSC/DECRC 衝突**: 子プロセスと save/restore が競合する可能性
- **exec_capture 同期ブロック**: コマンド実行中イベントループ停止
- **SIGWINCH**: 毎回 tty_size ioctl で検出（十分軽量）

## codex レビュー対応

| 指摘 | 対応 |
|---|---|
| ANSI/UTF-8 途中分割 | 子出力+オーバーレイを結合して単一 write |
| poll timeout 未発火 | monotonic_ms で時刻管理、poll_once 先頭でチェック |
| pty_spawn の sh -c 問題 | `shimux_pty_spawn_argv` を新規追加、argv 直接渡し |
| リサイズ伝播 | sig_setup_winch で子PTY同期済み + overlay.update_size() |

## PoC 実施後の知見

### --position-mode による3方式の比較

実装: `--position-mode simple/decstbm/cell`（default: simple）

| 方式 | 原理 | 利点 | 限界 |
|---|---|---|---|
| simple | DECSC+CUP で絶対座標に描画 | 任意位置指定可能 | 子プロセスの ANSI 操作全般に無防備 |
| decstbm | スクロール領域制限 + 端に固定描画 | スクロールから保護される | alt screen で崩壊、full-width 固定 |
| cell | VT パース + セルバッファ + 差分描画 | 完全なオーバーレイ | 未実装（shimux 本体の仕事） |

### simple モードの具体的な崩壊パターン

子プロセスの ANSI エスケープシーケンスがオーバーレイ座標上のセルに影響する:

- **DCH** (`ESC[P`, 文字削除): オーバーレイ文字が左にシフト → ゴミ表示が累積
- **EL** (`ESC[K`, 行消去): オーバーレイが消える
- **ICH** (`ESC[@`, 文字挿入): オーバーレイが右にずれる
- **スクロール**: オーバーレイが上に流れて残る

実例: `top` コマンド実行中に right 配置のオーバーレイが `hhello helhello` のように崩壊。
原因は `top` が CPU 表示行を DCH+再描画で更新し、桁数変動でズレが蓄積するため。

### decstbm モードの限界

- **Alternate Screen Buffer** (`ESC[?1049h`): `top`, `vim` 等の全画面アプリが使用。
  alt screen に入ると DECSTBM がリセットされ、子プロセスが画面全体に描画可能になる。
- **--top 配置**: 全画面コマンドが row 1 から描画するため overlay が上書きされる。
  子 PTY サイズは height 分少なく設定済みだが、alt screen 内では無意味。
- **--bottom 配置**: 比較的安全。多くのコマンドは下端まで描画しないため。

### shimux 本体への設計指針

完全なオーバーレイには **入れ子 PTY + セルグリッド管理** が必須:

```
real terminal (rows=N)
  └─ shimux
       ├─ overlay 領域（直接描画）
       └─ inner PTY (rows=N-overlay_height)
            └─ 子プロセス（inner PTY 内で完結）
```

- 子プロセス出力を VT パースしてセルバッファに描画
- オーバーレイをセルバッファ上に上書き
- 差分を実端末に出力
- これは tmux のペイン管理と本質的に同じアーキテクチャ
