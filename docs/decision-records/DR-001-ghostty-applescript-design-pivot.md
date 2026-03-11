# DR-001: Ghostty AppleScript 対応による設計転換

## 問題/背景

shimux は Ghostty 上で Claude Code Agent Teams の tmux モードを動作させる tmux 互換 CLI ラッパーである。
当初の設計では、ターミナルアプリを外部から操作するために以下の複雑なアーキテクチャが必要だった:

- **gmux-agent**: 各ペインに常駐するエージェントプロセス
- **PTY プロキシ**: 疑似端末の中継層
- **Unix ソケット通信**: エージェント間の通信基盤
- **キーシミュレーション**: アクセシビリティ権限を用いたキー入力送信

この設計は PoC (poc001-go, poc002-pty-proxy, poc003-tty-overlay) で検証されたが、
複雑性が高く、アクセシビリティ権限の要求がユーザー体験を損なう課題があった。

## 決定内容

Ghostty 1.3.0 (2026年3月) で AppleScript サポートがプレビュー機能として追加されたことを受け、
アーキテクチャを「tmux CLI を受けて osascript で Ghostty の AppleScript API を呼ぶだけの薄いラッパー」に全面転換する。

### 不要になったもの

- gmux-agent（各ペインへの常駐プロセス）
- PTY プロキシ（疑似端末の中継）
- Unix ソケット通信（プロセス間通信基盤）
- キーシミュレーション（アクセシビリティ権限によるキー入力）
- アクセシビリティ権限の要求

### AppleScript で可能な操作

- split（ペイン分割）
- focus（ペインフォーカス移動）
- close（ペイン閉じ）
- input text（テキスト入力）
- send key（キー送信）
- perform action（アクション実行）
- list terminals（id/name/cwd の一覧取得）

## 選択理由/トレードオフ

### 採用理由

- アーキテクチャが劇的に単純化される（常駐プロセス不要、ソケット通信不要）
- アクセシビリティ権限が不要になり、ユーザー体験が改善
- Ghostty 公式 API に依存するため、ハック的なアプローチが不要
- Rust FFI が最小限で済む（osascript 呼び出しとファイル I/O 程度）

### トレードオフ・制限

- **ペインサイズ取得不可**: 行x列でのサイズ取得ができない
- **PID/TTY 取得不可**: ペイン内プロセスの PID や TTY パスを取得できない
- **リサイズ制限**: ピクセル単位のみ（行x列での指定不可）
- **API 安定性**: AppleScript API はプレビュー機能であり、Ghostty 1.4 で破壊的変更の可能性がある
- **macOS 限定**: AppleScript は macOS 固有技術

## 影響

- PoC コード (poc001-go, poc002-pty-proxy, poc003-tty-overlay) と旧設計書は archive@ ブランチに移動
- MoonBit + Rust FFI の技術スタックは継続するが、FFI の範囲が大幅に縮小
- tmux コマンド互換性の要件（Claude Code が発行する tmux コマンドのリスト等）は PoC から引き継ぐ
