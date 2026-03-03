# shimux MoonBit PoC ワークスペース

## 目的

shimux（tmux 互換 CLI ラッパー）を MoonBit + Rust FFI で実装する PoC。
tty/pty 制御の技術検証から始め、現在は本実装（poc3）に移行中。

## shimux とは

Ghostty 等のターミナルエミュレータ上で tmux 依存ツール（Claude Code Agent Teams 等）を動作させるための **tmux 互換 CLI ラッパー**。

## ディレクトリ構成

```
wip-moonbit-pty-poc/
├── CONTEXT.md              # このファイル
├── docs/
│   ├── design/             # 設計書・設計判断
│   │   ├── DESIGN.md       # poc3 設計書（アーキテクチャ・FFI・Phase計画）
│   │   └── poc-comparison.md # poc/poc2/poc3 比較と設計経緯
│   ├── knowledge/          # 技術知見
│   │   ├── moonbit-rust-ffi-findings.md  # FFI 実装で得た知見
│   │   └── moonbit-project-guide.md      # MoonBit プロジェクトガイド
│   └── research/           # 調査資料
│       ├── fd-io-flow.md               # FD/IO フロー設計
│       ├── ipc-research.md             # IPC 方式調査（Unix socket）
│       ├── mouse-clipboard-imgcat.md   # マウス/クリップボード/imgcat
│       └── tty-initial-design.md       # TTY 初期設計
├── poc/                    # 初代 PoC（参照用、動作確認済み）
└── poc3/                   # 現在の実装（★ここが本体）
    ├── ffi/src/            # Rust FFI (C ABI)
    ├── lib/ffi/            # MoonBit バインディング
    ├── lib/agent/          # イベントループ（実装中）
    ├── lib/observer/       # Observer trait
    └── cmd/agent/          # shimux-agent エントリーポイント
```

## 技術スタック

```
MoonBit (native backend) — ロジック層
    └── extern "C" → Rust staticlib (C ABI) — システム層
                          └── libc クレート → POSIX API
```

## 現在の進捗

- **Phase 1 FFI 層**: 完了（Rust 24関数 + MoonBit バインディング、テスト 27件全パス）
- **Phase 1 イベントループ**: 実装完了（テスト 38件全パス）
- Phase 計画の詳細: `docs/design/DESIGN.md`

### ビルド・テスト

```bash
# Rust FFI ビルド（初回・変更時）
(cd poc3/ffi && cargo build --release)

# チェック
(cd poc3 && moon check --target native)

# テスト（38件）
(cd poc3 && moon test --target native)

# ビルド（メインパッケージ指定が必須）
(cd poc3 && moon build cmd/agent --target native)

# 実行
poc3/_build/native/debug/build/cmd/agent/agent.exe
```

**注意**: `moon build --target native`（パッケージ未指定）は使えない。lib パッケージ（lib/ffi, lib/agent）にテスト用の `cc-link-flags` があるため、exe としてリンクされ `_main` 未定義エラーになる。`moon build cmd/agent --target native` でメインパッケージのみをビルドすること。

## 関連リソース

### shimux リポジトリ内

- `go-poc/` ワークスペース — Go による PoC 実装（参照用）
- `wip-redesign/` ワークスペース — 設計作業

### sandbox-moonbit リポジトリ

パス: `~/.local/share/repos/github.com/kawaz/sandbox-moonbit/`

- `wip-cli-parser/` — CLI パーサの設計（Phase 1 型設計完了、実装コード 0 行）
- `default/docs/shimux-analysis.md` — MoonBit 移植検討資料
