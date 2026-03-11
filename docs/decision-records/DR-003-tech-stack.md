# DR-003: 技術スタック

## 問題/背景

shimux の実装言語とビルドターゲットを決定する必要がある。
PoC フェーズでは Go (poc001)、MoonBit + Rust FFI (poc002, poc003) の両方を検証した。

## 決定内容

**MoonBit (native backend) + 最小限の Rust FFI** を採用する。

### MoonBit native backend

- MoonBit の native backend でネイティブバイナリを生成
- `moon build --target native` でビルド
- extern "C" で C 関数（libc の popen/pclose 等）を呼び出し可能

### Rust FFI の範囲

以下の最小限の用途に限定:

- **osascript 実行**: libc の popen/pclose、または posix_spawn を使用
- **ファイル I/O**: 設定ファイルの読み書き等

### 不要になった FFI

PoC 時代に必要だった以下の Rust FFI は AppleScript 対応により全て不要:

- PTY 操作 (openpty, forkpty)
- Unix ソケット (AF_UNIX)
- シグナルハンドリング (SIGWINCH 等)
- termios 操作

## 選択理由/トレードオフ

### MoonBit を選択した理由

- PoC で MoonBit の開発体験を確認済み
- native backend で十分な実行性能
- 型安全性と表現力（enum, pattern matching）
- kawaz/kuu.mbt (CLI パーサライブラリ) との親和性

### Go を不採用とした理由

- poc001-go で Go 実装を検証したが、MoonBit の方が型表現力に優れる
- MoonBit エコシステムへの貢献を優先

### Rust FFI を最小限にする理由

- AppleScript 対応により複雑な FFI が不要になった
- osascript 実行は MoonBit の extern "C" から直接 libc を呼ぶだけでも実現可能
- FFI 境界が少ないほどデバッグ・保守が容易

## 影響

- ビルドは 2 段階: Rust FFI (`cargo build`) + MoonBit (`moon build --target native`)
- justfile でビルドステップを統合管理
- MoonBit の native backend はまだ成熟途上のため、コンパイラの制約に注意が必要
